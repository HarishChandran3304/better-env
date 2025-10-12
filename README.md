# better-env

Encrypted secrets, zero plaintext, instant runtime loading.

better-env is a simple, secure CLI tool to manage environment variables centrally on your machine using PGP encryption (via ProtonMail's gopenpgp). Keep your secrets out of repos and out of plaintext files, loading them into apps only at runtime. Never accidentally commit secrets again. `.better-env` files are fully commit-safe.

## Why better-env?
- Global encrypted store for all your secrets (per user)
- No plaintext .env files needed in projects
- Easy per-project selection of which keys to load
- Shell-friendly: export/unset to the parent shell, or run commands in an isolated child environment

## Quickstart
### Installation

- Script (macOS/Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/HarishChandran3304/better-env/main/scripts/install.sh | sh
```

- Go install (devs):

```bash
go install github.com/HarishChandran3304/better-env@latest
mv ~/go/bin/bnv /usr/local/bin/bnv # ensure on PATH
```

- Manual download:
  - Download the latest release archive for your OS/ARCH from the Releases page
  - Verify the checksum using `shasum -a 256` and `checksums.txt`
  - Extract and place `bnv` into a directory on your `PATH`

### Setup

```bash
bnv setup
```

### Usage

#### 1) Store secrets (one-time)

```bash
bnv store KEY
```

#### 2) Import and use secrets

```bash
cd /path/to/my/project
bnv init
bnv add KEY
bnv load
```

### Shell Completions

Generate completions and install for your shell:

```bash
# bash
bnv completion bash > ~/.local/share/bash-completion/bnv || sudo tee /etc/bash_completion.d/bnv > /dev/null

# zsh
bnv completion zsh > "${fpath[1]}/_bnv" 2>/dev/null || bnv completion zsh > ~/.zsh/completions/_bnv

# fish
bnv completion fish > ~/.config/fish/completions/bnv.fish

# powershell
bnv completion powershell | Out-String | Invoke-Expression
```

## Platforms supported

better-env officially supports Unix-like shells (macOS, Linux). Windows is partially supported: all core commands work, but `load`/`unload` shell integration is POSIX-only and won’t affect PowerShell/cmd environments.

## Learn More

See the full documentation [here](https://better-env.dev/docs)
