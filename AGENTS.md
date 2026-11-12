# Agents

Guidance for any coding agent working in this repository.

- Agent contract: [CLAUDE.md](CLAUDE.md)
- Estate conventions: https://github.com/JorisJonkers-dev/workspace/blob/main/CLAUDE.md

A Go service with a web frontend in `web/`. The toolchain is pinned in `mise.toml` (`mise install`); tasks are defined in `Taskfile.yml`.

## Build and test

- Full local gate (everything CI runs on a pull request except E2E): task check
- Tests: task test
- Lint: task lint
- Build: task build
- Regenerate code and check for drift: task gen:check
