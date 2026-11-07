# notes-sync tooling

Keeps the notes vault (org-roam, plain git) in sync between the Macs, the
homelab bare repo and the Life-Base app. Architecture and rollback live in
`docs/NOTES_SYNC.md`; this file is the per-machine setup.

| Piece | Where it runs | What it does |
| --- | --- | --- |
| `notes-autosync.sh` | Mac1, Mac2 (optionally Linux/Windows) | debounced `add/commit/pull --rebase --autostash/push` on `main` |
| `com.example.notes-autosync.plist` | macOS launchd | runs the script on file changes and every 5 min |
| `worker/` | homelab (systemd) | git <-> API sync, outbound HTTPS only |
| `../../deploy/nixos/notes-git.nix` | homelab (DRAFT) | bare repo, SSH, GitHub mirror, worker unit |

## Safety rules the script enforces

* only acts when `main` is checked out (`NOTES_BRANCH` to change);
* does nothing during a rebase, merge, cherry-pick, revert or bisect;
* skips if `.git/index.lock` exists;
* waits for the tree to be idle (`--debounce`, default 300 s);
* never force-pushes; on a conflict it runs `git rebase --abort`, notifies
  (macOS notification, else stderr, at most once per hour) and exits 1;
* kill switch: `touch <repo>/.notes-autosync-ignore` pauses it, delete the
  file to resume. The state/lock files live in `.git/`, never in the work tree.
* Logs: `~/.local/state/notes-autosync/autosync.log` (rotated at 1 MB).

The "AI worktree dirty" guard from the plan is intentionally not needed: AI
work lives in a separate worktree (`~/personal/notes-ai`), which `git add -A`
in the main worktree can never sweep. See `docs/NOTES_SYNC.md`.

## Mac1 / Mac2

1. Make sure the clone has an upstream and SSH works non-interactively:
   ```sh
   git clone git@HOMELAB-TAILSCALE-NAME:notes.git ~/personal/notes   # or git remote set-url origin ...
   cd ~/personal/notes && git branch -u origin/main && git pull
   ```
   Generate a **per-device** key (`ssh-keygen -t ed25519 -f ~/.ssh/notes_homelab -C "notes mac1"`),
   add its public key to `users.users.git.openssh.authorizedKeys.keys` in
   `notes-git.nix`, and add a `Host` entry in `~/.ssh/config` with
   `IdentityFile ~/.ssh/notes_homelab` and `IdentitiesOnly yes`.
2. Install the script: `mkdir -p ~/bin && cp notes-autosync.sh ~/bin/ && chmod +x ~/bin/notes-autosync.sh`.
3. Dry run: `~/bin/notes-autosync.sh --repo ~/personal/notes --now` then `tail ~/.local/state/notes-autosync/autosync.log`.
4. launchd: copy `com.example.notes-autosync.plist` to
   `~/Library/LaunchAgents/com.<you>.notes-autosync.plist`, replace `YOURUSER`
   and paths, `touch ~/.notes-autosync.trigger`, then
   `launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.<you>.notes-autosync.plist`.
5. Add the `.gitignore` entries below in the vault (on an `ai/chore/` branch,
   reviewed by you) **before** the first run so derived files are not committed.
6. Repeat on the second Mac with its own key. The very first sync on a machine
   with local edits: run with `--now` while watching the log.

Alternative to launchd: `fswatch -o ~/personal/notes/second-brain | while read; do ~/bin/notes-autosync.sh --repo ~/personal/notes --wait & done`
(the lock makes overlapping runs harmless), or `--loop` under tmux.

## Editors: touch the trigger file on save

Both hooks only `touch` a file; launchd (via `WatchPaths`) does the rest, and
the debounce still applies.

Neovim (`init.lua`):

```lua
vim.api.nvim_create_autocmd("BufWritePost", {
  pattern = { "*/personal/notes/*.org", "*/personal/notes/*.md" },
  callback = function()
    local f = io.open(vim.fn.expand("~/.notes-autosync.trigger"), "w")
    if f then f:write(os.time()) f:close() end
  end,
})
```

Emacs (`init.el`):

```elisp
(defun my/notes-autosync-touch ()
  (when (and buffer-file-name
             (string-match-p "/personal/notes/" buffer-file-name))
    (write-region (number-to-string (float-time)) nil
                  (expand-file-name "~/.notes-autosync.trigger") nil 'silent)))
(add-hook 'after-save-hook #'my/notes-autosync-touch)
```

## Windows (optional)

The PWA is the Windows client; a local clone is optional. If wanted: install
Git for Windows (it ships bash 5 and `ssh`), clone the repo, then create a Task
Scheduler task: trigger "At log on" repeating every 5 minutes, action
`"C:\Program Files\Git\bin\bash.exe" -lc "~/bin/notes-autosync.sh --repo /c/Users/YOU/notes"`,
"Run only when user is logged on". `stat`/`date` behave like Linux there.
Not tested.

## Proposed `.gitignore` additions (for the vault repo)

```gitignore
# derived / machine-local
org-roam.db
*.db-journal
.DS_Store
.#*
\#*#
*~
*.swp
*.swo
.org-id-locations
auto-save-list/
.emacs.d/
.obsidian/workspace*.json
*.orgtmp
.notes-autosync-ignore
# AI worktree lives outside the repo, but be safe if one is created inside
/notes-ai/
```

## The homelab worker

See `worker/main.go` header for the environment variables. Build/test with
`go build ./tools/notes-sync/worker && go test ./tools/notes-sync/worker/...`
from the repo root (not run in the authoring sandbox). One-off run for
debugging: `API_URL=... VAULT_SYNC_TOKEN=... REPO_PATH=/path/to/clone ./worker -once`.
