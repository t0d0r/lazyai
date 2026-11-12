# 2026-11-12 - gitleaks pre-commit hook

## Done

- `.githooks/pre-commit` scans the staged changes with
  `gitleaks git --pre-commit --staged --redact` and refuses the commit
  on a finding, or when gitleaks is not installed.
- `make hooks` turns it on for a clone
  (`git config core.hooksPath .githooks`); it is on in the working
  clone.
- Checked: a staged file with a made-up GitHub token is refused with
  the rule, file and line shown and the token redacted; without
  gitleaks on PATH the hook exits 1 with an install hint; a clean
  commit passes.
- specs/release.md (Security), README (Build), todo updated.

## Decisions

- Tracked `.githooks` plus a make target, not a file in `.git/hooks`:
  versioned, same hook in every clone.
- Fail closed without gitleaks; `--no-verify` is the way around.

## Open

- A new clone needs `make hooks` once; nothing enforces it. The scan
  before each push stays in the release steps.
- Not released; seven commits wait for v0.10.1.

## Files

.githooks/pre-commit, Makefile, specs/{release.md,README.md},
README.md, todo/todo.md.

## Run and verify

`make hooks`; stage a file with a fake token and `git commit`: refused.
`make check` is unchanged.

## Next step

Next open todo item, or release v0.10.1 (specs/release.md).
