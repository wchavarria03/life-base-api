// Command worker is the homelab side of the notes vault sync (see
// docs/NOTES_SYNC.md). It is outbound-only: it talks to the Render API over
// HTTPS and to a local git work tree backed by the bare repo, and never needs
// to be reachable from the internet.
//
// One cycle:
//  1. refresh the work tree from the bare repo (fetch + hard reset to origin);
//  2. diff since the last synced commit and POST changed .org/.md files to
//     /v1/internal/notes/upsert;
//  3. GET /v1/internal/notes/pending, write those files, commit, push
//     (never force), then POST /v1/internal/notes/ack for each;
//  4. POST /v1/internal/notes/sync-done when something changed or the user
//     pressed Sync in the app.
//
// Configuration (environment):
//
//	API_URL            base URL of the API, e.g. https://ledger-api-o3fa.onrender.com (required)
//	VAULT_SYNC_TOKEN   service token (required)
//	REPO_PATH          git work tree cloned from the bare repo (required)
//	GIT_BRANCH         branch to sync (default main)
//	GIT_REMOTE         remote name in the work tree (default origin)
//	STATE_FILE         last-synced-commit file (default $REPO_PATH/.git/notes-sync-last-commit)
//	TRIGGER_FILE       optional; its mtime changing (post-receive hook) wakes the worker
//	POLL_INTERVAL      how often to poll the API in daemon mode (default 60s)
//	GIT_AUTHOR_NAME / GIT_AUTHOR_EMAIL  identity for app-written commits
//
// Flags: -once runs a single cycle and exits (for cron/systemd timers).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type config struct {
	APIURL       string
	Token        string
	RepoPath     string
	Branch       string
	Remote       string
	StateFile    string
	TriggerFile  string
	PollInterval time.Duration
	GitName      string
	GitEmail     string
}

func loadConfig() (*config, error) {
	cfg := &config{
		APIURL:      strings.TrimRight(os.Getenv("API_URL"), "/"),
		Token:       os.Getenv("VAULT_SYNC_TOKEN"),
		RepoPath:    os.Getenv("REPO_PATH"),
		Branch:      envOr("GIT_BRANCH", "main"),
		Remote:      envOr("GIT_REMOTE", "origin"),
		StateFile:   os.Getenv("STATE_FILE"),
		TriggerFile: os.Getenv("TRIGGER_FILE"),
		GitName:     envOr("GIT_AUTHOR_NAME", "notes-sync"),
		GitEmail:    envOr("GIT_AUTHOR_EMAIL", "notes-sync@localhost"),
	}
	if cfg.APIURL == "" || cfg.Token == "" || cfg.RepoPath == "" {
		return nil, errors.New("API_URL, VAULT_SYNC_TOKEN and REPO_PATH are required")
	}
	u, err := url.Parse(cfg.APIURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("API_URL is not a valid URL: %q", cfg.APIURL)
	}
	if u.Scheme != "https" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
		return nil, errors.New("API_URL must be https (the service token is a bearer credential)")
	}
	if cfg.StateFile == "" {
		cfg.StateFile = filepath.Join(cfg.RepoPath, ".git", "notes-sync-last-commit")
	}
	poll := envOr("POLL_INTERVAL", "60s")
	cfg.PollInterval, err = time.ParseDuration(poll)
	if err != nil || cfg.PollInterval < 5*time.Second {
		return nil, fmt.Errorf("POLL_INTERVAL must be a duration >= 5s, got %q", poll)
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type worker struct {
	cfg  *config
	http *http.Client
}

func main() {
	os.Exit(run())
}

// run is main's body so deferred cleanup runs before the process exits with
// a non-zero code (log.Fatal inside main would skip the defers).
func run() int {
	once := flag.Bool("once", false, "run a single sync cycle and exit")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		log.Printf("config: %v", err)
		return 1
	}
	w := &worker{cfg: cfg, http: &http.Client{Timeout: 60 * time.Second}}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *once {
		if err := w.cycle(ctx); err != nil {
			log.Printf("cycle failed: %v", err)
			return 1
		}
		return 0
	}
	w.daemon(ctx)
	return 0
}

// daemon runs a cycle at start, whenever the trigger file changes (checked
// every 2 s, a local stat), and every PollInterval (to pick up app edits and
// Sync-button requests from the API).
func (w *worker) daemon(ctx context.Context) {
	log.Printf("notes-sync worker started (repo=%s api=%s poll=%s)", w.cfg.RepoPath, w.cfg.APIURL, w.cfg.PollInterval)
	lastTrigger := w.triggerMTime()
	nextPoll := time.Now()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		due := !time.Now().Before(nextPoll)
		if mt := w.triggerMTime(); !mt.Equal(lastTrigger) {
			lastTrigger = mt
			due = true
		}
		if due {
			if err := w.cycle(ctx); err != nil {
				log.Printf("cycle failed: %v", err)
			}
			nextPoll = time.Now().Add(w.cfg.PollInterval)
		}
		select {
		case <-ctx.Done():
			log.Printf("shutting down")
			return
		case <-tick.C:
		}
	}
}

func (w *worker) triggerMTime() time.Time {
	if w.cfg.TriggerFile == "" {
		return time.Time{}
	}
	fi, err := os.Stat(w.cfg.TriggerFile)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// ---------------------------------------------------------------- git

func (w *worker) gitBytes(ctx context.Context, args ...string) ([]byte, error) {
	full := append([]string{"-C", w.cfg.RepoPath, "-c", "user.name=" + w.cfg.GitName, "-c", "user.email=" + w.cfg.GitEmail}, args...)
	cmd := exec.CommandContext(ctx, "git", full...) //nolint:gosec // fixed "git" binary, no shell; args are internal constants plus validated work-tree paths
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func (w *worker) git(ctx context.Context, args ...string) (string, error) {
	out, err := w.gitBytes(ctx, args...)
	return strings.TrimSpace(string(out)), err
}

func (w *worker) head(ctx context.Context) (string, error) {
	return w.git(ctx, "rev-parse", "HEAD")
}

func (w *worker) readState() string {
	b, err := os.ReadFile(w.cfg.StateFile) //nolint:gosec // StateFile comes from operator config (env), not request input
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if !validCommit(s) {
		return ""
	}
	return s
}

func (w *worker) writeState(commit string) error {
	return os.WriteFile(w.cfg.StateFile, []byte(commit+"\n"), 0o600)
}

// refresh makes the work tree an exact mirror of the remote branch. Any
// unpushed local commit from a failed earlier cycle is discarded on purpose:
// its notes are still pending_write in the API and get rewritten next cycle.
func (w *worker) refresh(ctx context.Context) error {
	if _, err := w.git(ctx, "fetch", "--quiet", w.cfg.Remote); err != nil {
		return err
	}
	if _, err := w.git(ctx, "checkout", "--quiet", "--force", "-B", w.cfg.Branch, w.cfg.Remote+"/"+w.cfg.Branch); err != nil {
		return err
	}
	_, err := w.git(ctx, "clean", "-fdq")
	return err
}

// changesSince returns the syncable changes between last and head. An unknown
// or non-ancestor last commit (first run, rewritten history) means a full resync.
func (w *worker) changesSince(ctx context.Context, last, head string) ([]change, error) {
	if last != "" {
		if _, err := w.git(ctx, "merge-base", "--is-ancestor", last, head); err == nil {
			if last == head {
				return nil, nil
			}
			out, err := w.gitBytes(ctx, "diff", "--name-status", "-z", "--no-renames", last, head)
			if err != nil {
				return nil, err
			}
			return parseNameStatusZ(out), nil
		}
		log.Printf("last synced commit %s is not an ancestor of %s: full resync", last, head)
	}
	out, err := w.gitBytes(ctx, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	return parseLsFilesZ(out), nil
}

// ---------------------------------------------------------------- API

func (w *worker) api(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, w.cfg.APIURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.cfg.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: http %d: %s", method, path, resp.StatusCode, truncate(string(data), 300))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

type upsertResponse struct {
	Results []struct {
		VaultPath string `json:"vault_path"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
	} `json:"results"`
}

type pendingNote struct {
	ID            string `json:"id"`
	VaultPath     string `json:"vault_path"`
	Content       string `json:"content"`
	ContentFormat string `json:"content_format"`
}

type pendingResponse struct {
	SyncRequested bool          `json:"sync_requested"`
	Notes         []pendingNote `json:"notes"`
}

// ---------------------------------------------------------------- cycle

func (w *worker) cycle(ctx context.Context) error {
	if err := w.refresh(ctx); err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	head, err := w.head(ctx)
	if err != nil {
		return err
	}
	if !validCommit(head) {
		return fmt.Errorf("unexpected HEAD %q", head)
	}

	// Phase 1: git -> API.
	changed, err := w.pushChanges(ctx, head)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	if err := w.writeState(head); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	// Phase 2: API -> git.
	var pending pendingResponse
	if err := w.api(ctx, http.MethodGet, "/v1/internal/notes/pending", nil, &pending); err != nil {
		return fmt.Errorf("pending: %w", err)
	}
	newHead := head
	if len(pending.Notes) > 0 {
		h, err := w.writeBack(ctx, pending.Notes)
		if err != nil {
			return fmt.Errorf("write back: %w", err)
		}
		if h != "" {
			newHead = h
			changed = true
			// Our own push will echo through the bare repo; moving the state
			// forward now keeps that echo from being re-upserted (and
			// misread as a conflicting vault edit).
			if err := w.writeState(newHead); err != nil {
				return fmt.Errorf("write state: %w", err)
			}
		}
	}

	if changed || pending.SyncRequested {
		if err := w.api(ctx, http.MethodPost, "/v1/internal/notes/sync-done", map[string]string{"commit": newHead}, nil); err != nil {
			return fmt.Errorf("sync-done: %w", err)
		}
	}
	return nil
}

// pushChanges sends git changes since the last synced commit to the API and
// reports whether there was anything to send.
func (w *worker) pushChanges(ctx context.Context, head string) (bool, error) {
	all, err := w.changesSince(ctx, w.readState(), head)
	if err != nil {
		return false, err
	}
	keep, skipped := filterChanges(all)
	if len(skipped) > 0 {
		log.Printf("skipped %d non-note change(s)", len(skipped))
	}
	var items []upsertItem
	for _, c := range keep {
		item := upsertItem{VaultPath: c.Path, ContentFormat: formatFor(c.Path), Commit: head}
		if c.Status == "D" {
			item.Deleted = true
			items = append(items, item)
			continue
		}
		full, err := safeJoin(w.cfg.RepoPath, c.Path)
		if err != nil {
			log.Printf("skip %q: %v", c.Path, err)
			continue
		}
		fi, err := os.Lstat(full)
		if err != nil || !fi.Mode().IsRegular() {
			continue // vanished or not a regular file (symlink, dir)
		}
		data, err := os.ReadFile(full) //nolint:gosec // full comes from safeJoin: confined to the work tree and checked to be a regular file
		if err != nil {
			log.Printf("skip %q: %v", c.Path, err)
			continue
		}
		if !isSyncableText(data) {
			log.Printf("skip %q: not syncable text", c.Path)
			continue
		}
		item.Content = string(data)
		items = append(items, item)
	}
	for _, batch := range batchItems(items, 50, 4<<20) {
		var resp upsertResponse
		if err := w.api(ctx, http.MethodPost, "/v1/internal/notes/upsert", map[string]any{"items": batch}, &resp); err != nil {
			return false, err
		}
		for _, r := range resp.Results {
			switch r.Status {
			case "error":
				log.Printf("upsert %s: error: %s", r.VaultPath, r.Error)
			case "conflict":
				log.Printf("upsert %s: CONFLICT flagged (app edit pending); resolve in the app", r.VaultPath)
			}
		}
	}
	return len(items) > 0, nil
}

// writeBack writes pending app edits into the work tree, commits and pushes
// them (plain push, never forced) and acks each note. It returns the new HEAD
// or "" if nothing changed on disk.
func (w *worker) writeBack(ctx context.Context, notes []pendingNote) (string, error) {
	var done []pendingNote
	for _, n := range notes {
		full, err := safeJoin(w.cfg.RepoPath, n.VaultPath)
		if err != nil {
			log.Printf("refusing pending note %s (%q): %v", n.ID, n.VaultPath, err)
			continue
		}
		if len(n.Content) > maxContentSize {
			log.Printf("refusing pending note %s: content too large", n.ID)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return "", err
		}
		if err := writeFileAtomic(full, []byte(n.Content)); err != nil {
			return "", err
		}
		done = append(done, n)
	}
	if len(done) == 0 {
		return "", nil
	}

	args := []string{"add", "--"}
	for _, d := range done {
		args = append(args, d.VaultPath)
	}
	if _, err := w.git(ctx, args...); err != nil {
		return "", err
	}
	head, err := w.head(ctx)
	if err != nil {
		return "", err
	}
	if _, err := w.git(ctx, "diff", "--cached", "--quiet"); err == nil {
		// Files already identical to git: nothing to commit, just ack.
		w.ackAll(ctx, done, head)
		return "", nil
	} else if !isExitCode(err, 1) {
		return "", err
	}

	msg := fmt.Sprintf("app: update %d note(s)", len(done))
	if _, err := w.git(ctx, "commit", "--quiet", "-m", msg); err != nil {
		return "", err
	}
	if _, err := w.git(ctx, "push", "--quiet", w.cfg.Remote, "HEAD:refs/heads/"+w.cfg.Branch); err != nil {
		// Typically a non-fast-forward: someone pushed meanwhile. Do NOT force.
		// The next cycle resets to the remote and rewrites the still-pending
		// notes; if the remote touched the same files, upsert flags a conflict.
		return "", fmt.Errorf("push rejected (will retry next cycle, never forced): %w", err)
	}
	newHead, err := w.head(ctx)
	if err != nil {
		return "", err
	}
	w.ackAll(ctx, done, newHead)
	return newHead, nil
}

func (w *worker) ackAll(ctx context.Context, notes []pendingNote, commit string) {
	for _, n := range notes {
		body := map[string]string{"id": n.ID, "commit": commit, "content_sha256": sha256Hex(n.Content)}
		if err := w.api(ctx, http.MethodPost, "/v1/internal/notes/ack", body, nil); err != nil {
			log.Printf("ack %s failed (will be rewritten next cycle): %v", n.ID, err)
		}
	}
}

func isExitCode(err error, code int) bool {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode() == code
	}
	return false
}

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".notes-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	// Note files are intentionally 0644, matching a normal git checkout.
	if err := os.Chmod(tmpName, 0o644); err != nil { //nolint:gosec // notes are meant to be world-readable like any git work-tree file
		cleanup()
		return err
	}
	return os.Rename(tmpName, path)
}
