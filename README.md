<p align="center">
<img width="120" height="120" alt="image" src="https://github.com/user-attachments/assets/3427bf56-07a6-4472-9ea6-faca2109b01d" />
</p>

<h1 align="center"> better-env </h1>

<h3 align="center"> A Secure, Developer-Friendly Alternative to .env </h3>

<p align="center">
Encrypted Secrets • Zero Plaintext • Instant Runtime Loading • 100% Commit Safe
</p>

better-env is a simple, secure CLI tool to manage environment variables centrally on your machine using PGP encryption. Keep your secrets out of repos and out of plaintext files, loading them into apps only at runtime. Never accidentally commit secrets again. `.better-env` files are fully commit-safe.

## Why better-env?
- Global encrypted store for all your secrets
- No plaintext .env files needed in projects
- Easy per-project selection of which keys to load
- Shell-friendly: start a subshell with secrets or run commands with injected secrets

## Quickstart
### Installation

- macOS/Linux:
```bash
curl -fsSL https://raw.githubusercontent.com/HarishChandran3304/better-env/main/scripts/install.sh | sh
```

- Windows (PowerShell):
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -Command "iwr -useb https://raw.githubusercontent.com/HarishChandran3304/better-env/main/scripts/install.ps1 | iex"
```
> [!NOTE]
> The Windows installer uses PowerShell to install, but after installation bnv can be run from any shell on your PATH.

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

# Interactive shell (recommended)
bnv shell
# ... your commands here ...
exit

# One-off command
bnv run python3 main.py
```

## Platforms supported

better-env officially supports Unix-like shells (macOS, Linux).

> [!WARNING]
> Windows support is experimental:
> 
> - `bnv run` works as expected.
> - `bnv shell` attempts to launch `pwsh`, then `powershell`, then `cmd.exe` for an interactive session with secrets. Shell startup semantics differ from POSIX shells, so profile scripts > may not run the same.
> - For the best experience on Windows, consider WSL or Git Bash/MSYS2.

## Contributing

Contributions welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md).

## Learn More

See the full documentation [here](https://better-env.dev/docs)
