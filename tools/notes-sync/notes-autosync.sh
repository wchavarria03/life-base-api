#!/usr/bin/env bash
# notes-autosync.sh - debounced auto-commit/pull/push for the notes vault.
#
# Intended to be triggered by launchd (WatchPaths + StartInterval), fswatch,
# or an editor hook, or run in --loop mode. Safe by construction:
#   - acts only when the checked-out branch is "main" (override: NOTES_BRANCH);
#   - does nothing during a rebase / merge / cherry-pick / revert / bisect;
#   - skips when .git/index.lock exists (an editor or git is busy);
#   - waits until the working tree has been idle for the debounce period;
#   - never force-pushes; on any conflict or failure it aborts the rebase
#     cleanly, notifies, and exits non-zero;
#   - honours a kill switch: create the file .notes-autosync-ignore in the
#     repo root (or $NOTES_AUTOSYNC_PAUSE_FILE) to pause everything.
#
# Compatible with bash 3.2 (stock macOS) and Linux.
#
# Usage: notes-autosync.sh [options] [REPO_DIR]
#   -r, --repo DIR       repository directory (default: $NOTES_REPO or cwd)
#   -d, --debounce SECS  idle time before committing (default 300)
#   -l, --log FILE       log file (default ~/.local/state/notes-autosync/autosync.log)
#       --loop           run forever (every --interval seconds) instead of once
#   -i, --interval SECS  loop interval (default 60)
#       --now            ignore the debounce (flush immediately)
#       --wait           instead of skipping while files are still changing,
#                        stay alive (up to 1 h) until the tree has been idle
#                        for the debounce period, then sync. Use this with
#                        launchd WatchPaths so the sync happens ~5 min after
#                        the LAST save rather than on the next timer tick.
#   -h, --help
#
# Exit codes: 0 ok / nothing to do / paused, 1 failure (conflict, push error), 2 usage.

set -u

REPO="${NOTES_REPO:-}"
DEBOUNCE=300
INTERVAL=60
LOOP=0
NOW=0
WAIT=0
BRANCH="${NOTES_BRANCH:-main}"
LOG="${NOTES_AUTOSYNC_LOG:-${XDG_STATE_HOME:-$HOME/.local/state}/notes-autosync/autosync.log}"
HOST="$(hostname -s 2>/dev/null || hostname)"

usage() {
  sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
  case "$1" in
    -r|--repo) REPO="${2:-}"; shift 2 ;;
    -d|--debounce) DEBOUNCE="${2:-}"; shift 2 ;;
    -l|--log) LOG="${2:-}"; shift 2 ;;
    -i|--interval) INTERVAL="${2:-}"; shift 2 ;;
    --loop) LOOP=1; shift ;;
    --now) NOW=1; shift ;;
    --wait) WAIT=1; shift ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "unknown option: $1" >&2; exit 2 ;;
    *) REPO="$1"; shift ;;
  esac
done

[ -n "$REPO" ] || REPO="$(pwd)"
case "$DEBOUNCE$INTERVAL" in
  *[!0-9]*) echo "debounce/interval must be integers" >&2; exit 2 ;;
esac
if ! cd "$REPO" 2>/dev/null; then
  echo "cannot cd to $REPO" >&2
  exit 2
fi
REPO="$(pwd)"
GITDIR="$(git rev-parse --git-dir 2>/dev/null)" || { echo "$REPO is not a git repository" >&2; exit 2; }
case "$GITDIR" in /*) ;; *) GITDIR="$REPO/$GITDIR" ;; esac
PAUSE_FILE="${NOTES_AUTOSYNC_PAUSE_FILE:-$REPO/.notes-autosync-ignore}"

mkdir -p "$(dirname "$LOG")" 2>/dev/null || true

# Never hang on a credential or host-key prompt in a background job.
export GIT_TERMINAL_PROMPT=0
if [ -z "${GIT_SSH_COMMAND:-}" ]; then
  export GIT_SSH_COMMAND="ssh -o BatchMode=yes -o ConnectTimeout=15"
fi

log() {
  # keep the log small: rotate once at ~1 MB
  if [ -f "$LOG" ] && [ "$(wc -c < "$LOG" | tr -d ' ')" -gt 1048576 ]; then
    mv "$LOG" "$LOG.1" 2>/dev/null || true
  fi
  printf '%s [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$HOST" "$*" >> "$LOG" 2>/dev/null || true
}

# notify MESSAGE - macOS notification if osascript exists, else stderr.
# Throttled to one notification per hour so a persistent failure running
# every few minutes does not spam.
notify() {
  msg="$1"
  log "NOTIFY: $msg"
  stamp="$GITDIR/notes-autosync.lastnotify"
  now="$(date +%s)"
  if [ -f "$stamp" ]; then
    last="$(cat "$stamp" 2>/dev/null || echo 0)"
    case "$last" in ''|*[!0-9]*) last=0 ;; esac
    if [ $((now - last)) -lt 3600 ]; then
      return 0
    fi
  fi
  echo "$now" > "$stamp" 2>/dev/null || true
  if command -v osascript >/dev/null 2>&1; then
    safe="$(printf '%s' "$msg" | tr -d '"\\' | cut -c1-200)"
    osascript -e "display notification \"$safe\" with title \"notes-autosync ($HOST)\"" >/dev/null 2>&1 || true
  else
    echo "notes-autosync: $msg" >&2
  fi
}

# mtime_of FILE - epoch seconds, portable between BSD (macOS) and GNU stat.
if stat -f %m / >/dev/null 2>&1; then
  mtime_of() { stat -f %m "$1" 2>/dev/null; }
else
  mtime_of() { stat -c %Y "$1" 2>/dev/null; }
fi

op_in_progress() {
  for f in rebase-merge rebase-apply MERGE_HEAD CHERRY_PICK_HEAD REVERT_HEAD BISECT_LOG; do
    if [ -e "$GITDIR/$f" ]; then
      echo "$f"
      return 0
    fi
  done
  return 1
}

# newest_dirty_mtime - newest mtime among modified/untracked/staged files
# (0 if none). Uses NUL-separated names so spaces/newlines are safe.
newest_dirty_mtime() {
  newest=0
  while IFS= read -r -d '' f; do
    [ -e "$f" ] || continue
    m="$(mtime_of "$f")"
    case "$m" in ''|*[!0-9]*) continue ;; esac
    [ "$m" -gt "$newest" ] && newest="$m"
  done < <({ git ls-files -m -o --exclude-standard -z; git diff --cached --name-only -z; } 2>/dev/null)
  echo "$newest"
}

abort_rebase_cleanly() {
  if [ -d "$GITDIR/rebase-merge" ] || [ -d "$GITDIR/rebase-apply" ]; then
    git rebase --abort >/dev/null 2>&1 || true
    log "rebase aborted"
  fi
}

run_once() {
  if [ -e "$PAUSE_FILE" ]; then
    log "paused ($PAUSE_FILE exists)"
    return 0
  fi

  cur="$(git symbolic-ref --short -q HEAD 2>/dev/null || true)"
  if [ "$cur" != "$BRANCH" ]; then
    log "skip: on branch '${cur:-detached}', not '$BRANCH'"
    return 0
  fi

  if op="$(op_in_progress)"; then
    log "skip: $op in progress"
    return 0
  fi
  if [ -e "$GITDIR/index.lock" ]; then
    log "skip: index.lock present"
    return 0
  fi

  # Single-instance lock (mkdir is atomic on every platform).
  lock="$GITDIR/notes-autosync.lock"
  if ! mkdir "$lock" 2>/dev/null; then
    oldpid="$(cat "$lock/pid" 2>/dev/null || echo 0)"
    if [ "$oldpid" != "0" ] && kill -0 "$oldpid" 2>/dev/null; then
      log "skip: another run is active (pid $oldpid)"
      return 0
    fi
    rm -rf "$lock"
    mkdir "$lock" 2>/dev/null || { log "skip: cannot take lock"; return 0; }
  fi
  echo $$ > "$lock/pid"
  trap 'rm -rf "$lock"' EXIT

  rc=0
  sync_body || rc=$?
  rm -rf "$lock"
  trap - EXIT
  return $rc
}

sync_body() {
  dirty="$(git status --porcelain 2>/dev/null | head -n 1)"
  if [ -n "$dirty" ]; then
    if [ "$NOW" -ne 1 ]; then
      waited=0
      while :; do
        newest="$(newest_dirty_mtime)"
        now="$(date +%s)"
        if [ "$newest" -le 0 ] || [ $((now - newest)) -ge "$DEBOUNCE" ]; then
          break
        fi
        if [ "$WAIT" -ne 1 ] || [ "$waited" -ge 3600 ]; then
          log "debounce: last change $((now - newest))s ago (< ${DEBOUNCE}s)"
          return 0
        fi
        sleep 15
        waited=$((waited + 15))
        # Re-check the guards that can change while we sleep.
        if [ -e "$PAUSE_FILE" ] || op_in_progress >/dev/null; then
          log "skip: paused or git operation started while waiting"
          return 0
        fi
        cur="$(git symbolic-ref --short -q HEAD 2>/dev/null || true)"
        if [ "$cur" != "$BRANCH" ]; then
          log "skip: branch changed to '${cur:-detached}' while waiting"
          return 0
        fi
      done
    fi
    if [ -e "$GITDIR/index.lock" ]; then
      log "skip: index.lock present"
      return 0
    fi
    git add -A >>"$LOG" 2>&1 || { notify "git add failed in $REPO"; return 1; }
    if ! git diff --cached --quiet; then
      n="$(git diff --cached --name-only | wc -l | tr -d ' ')"
      if ! git commit -q -m "auto($HOST): $n files" >>"$LOG" 2>&1; then
        notify "git commit failed in $REPO"
        return 1
      fi
      log "committed $n files"
    fi
  fi

  if ! git pull --rebase --autostash -q >>"$LOG" 2>&1; then
    abort_rebase_cleanly
    notify "pull --rebase failed (conflict?) in $REPO - resolve manually; autosync stopped for this run"
    return 1
  fi

  if [ "$(git rev-list --count '@{upstream}..HEAD' 2>/dev/null || echo 0)" != "0" ]; then
    if ! git push -q >>"$LOG" 2>&1; then
      # Someone pushed between pull and push: one retry, still never forced.
      if git pull --rebase --autostash -q >>"$LOG" 2>&1 && git push -q >>"$LOG" 2>&1; then
        log "pushed after retry"
        return 0
      fi
      abort_rebase_cleanly
      notify "push failed in $REPO"
      return 1
    fi
    log "pushed"
  fi
  return 0
}

if [ "$LOOP" -eq 1 ]; then
  log "loop mode started (interval ${INTERVAL}s, debounce ${DEBOUNCE}s, repo $REPO)"
  while :; do
    run_once || true
    sleep "$INTERVAL"
  done
fi

run_once
exit $?
