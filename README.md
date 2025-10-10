better-env
===========

Encrypted secrets, zero plaintext, instant runtime loading.

better-env is a simple, secure CLI tool to manage environment variables centrally on your machine using PGP encryption (via ProtonMail's gopenpgp). Keep your secrets out of repos and out of plaintext files, loading them into apps only at runtime. Never accidentally commit secrets again - .better-env files are fully commit-safe.

Platform support
----------------

better-env officially supports Unix-like shells (macOS, Linux). Windows is partially supported: all core commands work, but `load`/`unload` shell integration is POSIX-only and won’t affect PowerShell/cmd environments. See [Windows Support](#windows-support) for details and workarounds.

Why better-env?
- Global encrypted store for all your secrets (per user)
- No plaintext .env files needed in projects
- Easy per-project selection of which keys to load
- Shell-friendly: export/unset to the parent shell, or run commands in an isolated child environment


Index
-----

- [Installation](#installation)
- [Setup](#setup)
- [Usage](#usage)
- [Core Concepts](#core-concepts)
- [Command Reference](#command-reference)
- [Migration Guide (.env → better-env)](#migration-guide)
- [Notes for teams](#notes-for-teams)
- [Roadmap](#roadmap)


Installation
------------

Prerequisites:
- Go 1.21+ (to build from source)
- Git 2.30+ (if building from a clone)
- Clipboard tools (Linux only): `xclip` or `xsel` for `bnv copy`
- GPG external tool is NOT required; better-env embeds GPG functionality using ProtonMail’s gopenpgp library.

Install from source:

```bash
go install github.com/HarishChandran3304/better-env@latest
```

This installs the `bnv` binary to your `GOBIN` (usually `~/go/bin`). Ensure it’s on your `PATH`.


Build from source (git clone):

```bash
git clone https://github.com/HarishChandran3304/better-env.git
cd better-env
go build
sudo mv better-env /usr/local/bin/bnv
```


Setup
-----

1) One-time setup (creates the encrypted store and a keypair):

```bash
bnv setup
```

During setup, a GPG keypair is generated (you will also be prompted for a passphrase) and your encrypted secrets store is created at:

- Store directory: `$(os.UserConfigDir)/better-env`
  - `config.json` – store metadata
  - `public.key`, `private.key` – your keypair (private key may be passphrase-protected)
  - `secrets.gpg` – encrypted JSON map of KEY -> VALUE

On macOS, `os.UserConfigDir()` is typically `~/Library/Application Support`; on Linux it’s usually `~/.config`.

2) Add the shell function so `bnv load` and `bnv unload` can export/unset in your current shell (skip this step if you are on Windows):


For zsh:
```bash
echo 'bnv() { if [ "$1" = "load" ]; then eval "$(command bnv load)"; elif [ "$1" = "unload" ]; then eval "$(command bnv unload)"; else command bnv "$@"; fi }' >> ~/.zshrc && source ~/.zshrc
```

For bash:
```bash
echo 'bnv() { if [ "$1" = "load" ]; then eval "$(command bnv load)"; elif [ "$1" = "unload" ]; then eval "$(command bnv unload)"; else command bnv "$@"; fi }' >> ~/.bashrc && source ~/.bashrc
```

For any other shell just add the following to your shell config file:
```bash
bnv() { if [ "$1" = "load" ]; then eval "$(command bnv load)"; elif [ "$1" = "unload" ]; then eval "$(command bnv unload)"; else command bnv "$@"; fi }
```

Usage
-----

1) Store secrets in the global store:
```bash
bnv store API_KEY
Enter value: sk_live_123...
```


2) Initialize a project and pick which keys it uses:

```bash
cd /path/to/your/project
bnv init
bnv add API_KEY                  # add the key to this project’s .better-env
bnv load                         # export vars into current shell (via the shell function)
# or
bnv run node server.js           # run a command with secrets only in the child process
```


Core Concepts
-------------

- Global store: A single encrypted store of all your secrets lives under `$(os.UserConfigDir)/better-env`.
- Project link: Each project has a `.better-env` JSON file that links to your store and lists which keys the project uses.
- Two ways to use secrets:
  - `bnv load` / `bnv unload`: export/unset in your current shell via the shell function wrapper
  - `bnv run`: run a command with secrets only available to the child process


Command Reference
-----------------

All commands are subcommands of `bnv`. Use `-h` with any command for help.

Setup and Configuration
- setup
  - usage: `bnv setup`
  - Description: Initialize better-env, generate a keypair, and create the encrypted store.

- init
  - usage: `bnv init [--path PATH|-p PATH] [--force|-f]`
  - Description: Create a `.better-env` file in the current (or specified) directory and link it to your global store.
  - Flags:
    - `--path, -p` (default `.`): directory to place `.better-env`
    - `--force, -f`: overwrite an existing `.better-env`


Managing Secrets (Global Store)
- store
  - usage: `bnv store KEY`
  - Description: Add or update `KEY` in the encrypted store. Reads value interactively; prompts for passphrase.

- update
  - usage: `bnv update KEY`
  - Description: Update the value for an existing `KEY` in the store. Reads new value interactively.

- delete
  - usage: `bnv delete KEY [KEY...]`
  - Description: Delete one or more keys from the global store.

- rename
  - usage: `bnv rename OLD_KEY NEW_KEY`
  - Description: Rename a key in the global store. Offers to update the current project’s `.better-env` if present.

- show
  - usage: `bnv show KEY [KEY2 KEY3 ...]`
  - Description: Decrypt and print the values of one or more secrets. Prompts for passphrase.

- copy
  - usage: `bnv copy KEY [KEY2 KEY3 ...]`
  - Description: Decrypt and copy secret value(s) to the clipboard. Values are not printed to the terminal.

- import
  - usage: `bnv import ENV_FILE_PATH`
  - Description: Parse a `.env` file and import all variables into the encrypted store. Optionally delete the `.env` and create a `.better-env` file.
  - Examples:
    - `bnv import ./.env`
    - `bnv import /abs/path/to/.env`


Managing Project Keys (.better-env)
- add
  - usage: `bnv add KEY1 [KEY2 KEY3 ...]`
  - Description: Add one or more key names to the current project’s `.better-env` so they’ll be loaded.

- remove
  - usage: `bnv remove KEY [KEY...]`
  - Description: Remove one or more key names from the current project’s `.better-env`. Does not delete from the global store.

- list
  - usage: `bnv list [--all|-a]`
  - Description: List keys for the current project. With `--all`, list all keys in the global store.


Loading and Running
- load
  - usage: `bnv load`
  - Description: Decrypt and output `export KEY='VALUE'` lines. Use via the shell function so they affect your current shell.
  - Example:
    - `bnv load`  (with the function wrapper installed, this will export into your current shell)

- unload
  - usage: `bnv unload`
  - Description: Output `unset KEY` lines for keys in the current project, or all keys if the project lists none.

- run
  - usage: `bnv run COMMAND [ARGS...]`
  - Description: Run a child process with the requested secrets set in its environment. Secrets are not exported to your parent shell.
  - Examples:
    - `bnv run node server.js`
    - `bnv run python3 main.py`


Migration Guide
---------------

Migrate an existing project that uses a `.env` file:

1) One-time on your machine:
```bash
bnv setup
```

2) In your project directory, import your existing `.env`:
```bash
cd /path/to/your/project
bnv import ./.env
```
- You’ll be prompted to delete the original `.env` (recommended) and to create a matching `.better-env` file listing the imported keys.

3) Load secrets for development:
```bash
bnv load
# or keep secrets isolated to the child process
bnv run your-app-command
```

Notes for teams
---------------

- Do not commit your `.env`. Each developer should run `bnv setup` locally and then `bnv init` or use `bnv import` in their clone.
- `.better-env` is commit-safe. It no longer stores user-specific paths and can be safely versioned.
- CI/CD usage: prefer `bnv run` to scope secrets to a single process.


Roadmap
-------

- [ ] Backup/restore helpers for the encrypted store
- [ ] Optional multi-profile/namespaces per project
- [ ] Key rotation and audit logging helpers
- [ ] Publish prebuilt binaries and a Homebrew formula
- [ ] Versioning for the global store?
- [ ] Password caching (with apt TTL)?
- [ ] Improve support for Windows
- [ ] Improve support for team-based collaboration and workflows


Windows Support
---------------

Status: partial

Works:
- `setup`, `init`, `store`, `update`, `delete`, `rename`, `show`, `copy`, `import`, `add`, `remove`, `list`, `run`

Limitations:
- `load` / `unload` print POSIX `export`/`unset` lines and won’t affect PowerShell/cmd parent shells.

Workarounds:
- Use Git Bash or WSL to enable `load`/`unload` via the shell function.
- Prefer `bnv run …` to execute your app with secrets in the child process environment on native Windows.

Planned:
- Native PowerShell integration.

