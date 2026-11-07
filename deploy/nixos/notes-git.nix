# =============================================================================
# DRAFT - NEVER BUILT OR EVALUATED. Written without a Nix toolchain.
# Review every line, run `nix-instantiate --parse` / `nixos-rebuild build`
# (not `switch`) first, and expect to fix option names and typos.
# =============================================================================
#
# modules/notes-git.nix - bare git remote for the notes vault + GitHub mirror +
# the sync worker that keeps the Life-Base API in step with git.
# See docs/NOTES_SYNC.md in the life-base-api repo for the architecture.
#
# Prerequisites (one-time, by hand - datasets and secrets are not declarative):
#   1. zfs create data/notes                     # becomes /srv/data/notes
#   2. sops secrets (secrets.yaml), all new:
#        notes_sync_token          # same value as VAULT_SYNC_TOKEN on Render
#        notes_github_deploy_key   # private key of a WRITE deploy key on the
#                                  #   private GitHub mirror repo
#        hc_notes_mirror_url       # healthchecks.io ping URL (no trailing slash)
#   3. Create the private GitHub repo and add the deploy key (write access).
#   4. Replace every PLACEHOLDER below (SSH public keys, GitHub repo, hashes).
#   5. Add `modules/notes-git.nix` to the imports of configuration.nix and add
#      `/srv/data/notes` to the paths list in modules/restic-backup.nix.
{ config, lib, pkgs, ... }:

let
  # --- layout ---------------------------------------------------------------
  dataDir     = "/srv/data/notes";                 # ZFS dataset data/notes (mirror + weekly snapshots)
  bareRepo    = "${dataDir}/notes.git";            # the remote every device pushes to
  triggerFile = "${dataDir}/sync-trigger";         # touched by post-receive; watched by worker + mirror
  stateDir    = "/var/lib/notes-sync";             # disposable: worker's work tree + state
  workTree    = "${stateDir}/work";                # git clone of bareRepo, owned by the worker

  # PLACEHOLDER: the private GitHub mirror (SSH URL).
  githubRemote = "git@github.com:YOUR_USER/notes-vault-mirror.git";

  # PLACEHOLDER: public API URL of the Life-Base backend.
  apiUrl = "https://ledger-api-o3fa.onrender.com";

  # --- the worker binary ----------------------------------------------------
  # Built from the life-base-api repo (tools/notes-sync/worker, stdlib only).
  # The repo's go.mod has other dependencies (gin, ...), so vendorHash must be
  # the real hash: set it to lib.fakeHash, build once, copy the hash Nix prints.
  notesSyncWorker = pkgs.buildGoModule {
    pname = "notes-sync-worker";
    version = "0.1.0";
    # PLACEHOLDER: pin the repo revision you deploy.
    src = pkgs.fetchFromGitHub {
      owner = "YOUR_USER";
      repo = "life-base-api";
      rev = "PLACEHOLDER_COMMIT";
      hash = lib.fakeHash;
    };
    vendorHash = lib.fakeHash;
    subPackages = [ "tools/notes-sync/worker" ];   # only this main package
    # The binary is named after the package dir ("worker"); rename for clarity.
    postInstall = "mv $out/bin/worker $out/bin/notes-sync-worker";
  };

  # --- post-receive hook ----------------------------------------------------
  # Runs as the `git` user inside the bare repo after every push. It must stay
  # tiny and never fail the push: it only touches the trigger file. The worker
  # and the mirror unit react to that file (no sudo / systemctl needed).
  postReceive = pkgs.writeShellScript "notes-post-receive" ''
    touch ${triggerFile} || true
    exit 0
  '';

  # --- healthchecks.io pings (same onSuccess/onFailure pattern as
  # modules/healthchecks.nix; a template unit keeps it to one definition) ----
  hcPing = suffix: {
    serviceConfig = {
      Type = "oneshot";
      DynamicUser = true;
      # LoadCredential lets the DynamicUser read a root-only sops secret.
      LoadCredential = "url:${config.sops.secrets.hc_notes_mirror_url.path}";
    };
    # $CREDENTIALS_DIRECTORY is set by systemd for LoadCredential.
    script = ''
      ${pkgs.curl}/bin/curl -fsS -m 10 --retry 3 "$(cat "$CREDENTIALS_DIRECTORY/url")${suffix}" >/dev/null
    '';
  };
in
{
  # ---------------------------------------------------------------------------
  # Users: a dedicated `git` account. Shell is git-shell, so SSH can only run
  # git-receive-pack / git-upload-pack (no interactive shell, no arbitrary
  # commands). Per-device keys: one line per device so one can be revoked alone.
  # ---------------------------------------------------------------------------
  users.groups.git = { };
  users.users.git = {
    isSystemUser = true;
    group = "git";
    home = dataDir;
    shell = "${pkgs.git}/bin/git-shell";
    openssh.authorizedKeys.keys = [
      # `restrict` disables port/agent/X11 forwarding and PTY for that key.
      "restrict ssh-ed25519 AAAA_PLACEHOLDER_MAC1 notes mac1"
      "restrict ssh-ed25519 AAAA_PLACEHOLDER_MAC2 notes mac2"
      # "restrict ssh-ed25519 AAAA_PLACEHOLDER_WINDOWS notes windows"
    ];
  };

  # NOTE: sshd rejects accounts with a locked password ("!") when PAM is off.
  # NixOS runs sshd with UsePAM = true by default so key login works; if login
  # fails with "User git not allowed because account is locked", set
  # `users.users.git.hashedPassword = "*";` (no valid password, but not locked).

  # Extra sshd restrictions for this user. Match blocks must come last in
  # sshd_config; NixOS appends extraConfig at the end, but check the result in
  # /etc/ssh/sshd_config after the first build.
  services.openssh.extraConfig = ''
    Match User git
      AllowTcpForwarding no
      AllowAgentForwarding no
      X11Forwarding no
      PermitTTY no
      PasswordAuthentication no
  '';

  # ---------------------------------------------------------------------------
  # Tailscale-only SSH (same per-interface firewall style as Postgres/Samba).
  # For this to mean anything, modules/ssh.nix must NOT open 22 globally:
  # set `services.openssh.openFirewall = false;` there (confirm you still have
  # a way in over Tailscale first, or you will lock yourself out).
  # ---------------------------------------------------------------------------
  networking.firewall.interfaces.tailscale0.allowedTCPPorts = [ 22 ];

  # ---------------------------------------------------------------------------
  # Directories (tmpfiles, same pattern as the backup directories).
  # ---------------------------------------------------------------------------
  systemd.tmpfiles.rules = [
    "d ${dataDir}  0750 git git -"
    "d ${stateDir} 0750 git git -"
  ];

  # ---------------------------------------------------------------------------
  # sops secrets. Owners matter: the process reading a secret must own it
  # (see the Prometheus/healthchecks gotcha in the infra notes).
  # restartUnits makes sops-nix restart the unit when the VALUE changes.
  # ---------------------------------------------------------------------------
  sops.secrets.notes_sync_token = {
    owner = "git";
    restartUnits = [ "notes-sync-worker.service" ];
  };
  sops.secrets.notes_github_deploy_key = {
    owner = "git";
    mode = "0400";
    restartUnits = [ "notes-github-mirror.service" ];
  };
  sops.secrets.hc_notes_mirror_url = { };   # read via LoadCredential (root-owned is fine)

  # One env file for the worker; the placeholder is substituted at activation.
  sops.templates."notes-sync-worker.env" = {
    owner = "git";
    content = ''
      VAULT_SYNC_TOKEN=${config.sops.placeholder.notes_sync_token}
    '';
    restartUnits = [ "notes-sync-worker.service" ];
  };

  # GitHub's host key, pinned so the mirror push never prompts and cannot be
  # MITM'd. PLACEHOLDER: paste the current ed25519 key from
  # https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/githubs-ssh-key-fingerprints
  programs.ssh.knownHosts."github.com" = {
    publicKey = "ssh-ed25519 AAAA_PLACEHOLDER_GITHUB_HOST_KEY";
  };

  # ---------------------------------------------------------------------------
  # 1) Bare repository (idempotent init, hooks, safety settings).
  # ---------------------------------------------------------------------------
  systemd.services.notes-git-init = {
    description = "Initialise the bare notes git repository";
    wantedBy = [ "multi-user.target" ];
    after = [ "local-fs.target" ];
    path = [ pkgs.git ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
      User = "git";
      Group = "git";
    };
    script = ''
      if [ ! -d ${bareRepo} ]; then
        git init --bare --initial-branch=main ${bareRepo}
      fi
      # Never accept history rewrites or branch deletion from any client:
      # combined with the clients' "no force push" rule this keeps git a safe
      # source of truth.
      git -C ${bareRepo} config receive.denyNonFastForwards true
      git -C ${bareRepo} config receive.denyDeletes true
      git -C ${bareRepo} config receive.fsckObjects true
      ln -sfn ${postReceive} ${bareRepo}/hooks/post-receive
    '';
  };

  # ---------------------------------------------------------------------------
  # 2) Private GitHub mirror. Triggered by the post-receive trigger file, plus
  # an hourly retry timer in case GitHub was unreachable. Plain push of all
  # branches/tags, never forced.
  # ---------------------------------------------------------------------------
  systemd.services.notes-github-mirror = {
    description = "Push the notes bare repo to the private GitHub mirror";
    after = [ "notes-git-init.service" "network-online.target" ];
    wants = [ "network-online.target" ];
    path = [ pkgs.git pkgs.openssh ];
    environment.GIT_SSH_COMMAND =
      "ssh -i ${config.sops.secrets.notes_github_deploy_key.path} -o IdentitiesOnly=yes -o BatchMode=yes";
    serviceConfig = {
      Type = "oneshot";
      User = "git";
      Group = "git";
    };
    script = ''
      git -C ${bareRepo} remote get-url github >/dev/null 2>&1 \
        || git -C ${bareRepo} remote add github ${githubRemote}
      git -C ${bareRepo} push github --all
      git -C ${bareRepo} push github --tags
    '';
    # healthchecks.io: ping on success, ping /fail on failure.
    onSuccess = [ "notes-hc-success.service" ];
    onFailure = [ "notes-hc-fail.service" ];
  };

  systemd.services.notes-hc-success = hcPing "";
  systemd.services.notes-hc-fail    = hcPing "/fail";

  systemd.paths.notes-github-mirror = {
    wantedBy = [ "multi-user.target" ];
    pathConfig = {
      PathModified = triggerFile;
      Unit = "notes-github-mirror.service";
    };
  };
  systemd.timers.notes-github-mirror = {
    wantedBy = [ "timers.target" ];
    timerConfig = {
      OnCalendar = "hourly";
      Persistent = true;
      Unit = "notes-github-mirror.service";
    };
  };

  # ---------------------------------------------------------------------------
  # 3) The sync worker (outbound HTTPS only; no listening ports).
  # Runs as `git` so it can push to the bare repo without extra permissions.
  # The git-shell login shell does not matter: systemd starts the process
  # directly. The service token is only readable by this user.
  # ---------------------------------------------------------------------------
  systemd.services.notes-sync-worker = {
    description = "Notes vault <-> Life-Base API sync worker";
    wantedBy = [ "multi-user.target" ];
    after = [ "network-online.target" "notes-git-init.service" ];
    wants = [ "network-online.target" ];
    requires = [ "notes-git-init.service" ];
    path = [ pkgs.git pkgs.openssh ];
    environment = {
      API_URL = apiUrl;
      REPO_PATH = workTree;
      GIT_BRANCH = "main";
      GIT_REMOTE = "origin";
      TRIGGER_FILE = triggerFile;
      POLL_INTERVAL = "60s";   # each poll wakes the Render free dyno; raise if that matters
      GIT_AUTHOR_NAME = "notes-sync";
      GIT_AUTHOR_EMAIL = "notes-sync@homelab";
      HOME = stateDir;          # git wants a HOME for its global config lookup
    };
    # Clone the work tree from the local bare repo on first start. The repo may
    # be empty until the first device pushes; the worker then simply waits.
    preStart = ''
      if [ ! -d ${workTree}/.git ]; then
        git clone ${bareRepo} ${workTree} || true
      fi
    '';
    serviceConfig = {
      User = "git";
      Group = "git";
      EnvironmentFile = config.sops.templates."notes-sync-worker.env".path;
      ExecStart = "${notesSyncWorker}/bin/notes-sync-worker";
      Restart = "always";
      RestartSec = 15;
      # Hardening: the worker needs the network, its state dir and the bare repo.
      NoNewPrivileges = true;
      ProtectSystem = "strict";
      ProtectHome = true;
      PrivateTmp = true;
      ReadWritePaths = [ stateDir dataDir ];
      RestrictAddressFamilies = [ "AF_INET" "AF_INET6" "AF_UNIX" ];
      CapabilityBoundingSet = "";
    };
    onFailure = [ "notes-hc-fail.service" ];
  };
}
