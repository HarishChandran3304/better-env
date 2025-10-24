# Contributing

Thanks for helping improve better-env. Keep it simple and focused.

## Quick start
- Prereqs: Go 1.24+, Git, macOS/Linux/Windows
- Fork the repo and create a branch from `main`
- Make focused changes (small, self-contained commits)

## Local setup
```bash
# Clone your fork
git clone https://github.com/<your-username>/better-env.git
cd better-env

# Build the CLI (local binary)
go build -o bnv ./...

# Run help
./bnv --help

# Optional: install to GOPATH/bin so it's on PATH
go install ./...
```

## Local checks
```bash
go fmt ./...
go vet ./...
go build ./...
go test ./...
```

## Open a PR
- Use a clear title and short description of the change and why
- Link related issues (e.g., Fixes #123)
- Keep PRs small and targeted

That’s it — thanks for contributing!
