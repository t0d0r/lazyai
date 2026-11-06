# 2026-11-06 - update: Homebrew tap check

## Done

- `lazyai update` on a Homebrew install reads the published tap formula
  (t0d0r/homebrew-tap, Formula/lazyai.rb on GitHub) after finding a
  newer release, and only offers what brew can install:
  - tap current: unchanged behavior;
  - tap not newer than the installed version: says the release is not
    in the tap yet, asks nothing, exit 0;
  - tap between: offers the tap's version;
  - formula unreadable: warns and goes on as before.
- update.FormulaVersion (version line, else url archive name),
  Client.TapVersion and a shared HTTPS-only get(); brewTarget in
  commands.go holds the decision.
- Tests: TestFormulaVersion, TestTapVersion (internal/update),
  TestBrewTarget (cli_test.go).
- Checked live: a 0.9.0 build under a Cellar path offers
  v0.9.0 -> v0.10.0 after reading the real formula.

## Decisions

- The formula on GitHub, not `brew info`: the local tap clone lags until
  `brew update`, and `brew upgrade` updates taps before building.
- An unreadable formula does not block the upgrade; brew still decides.

## Open

- Not released; v0.10.0 does not have this check.
- The raw file host may serve a cached formula for a few minutes after
  a tap push.

## Files

internal/update/{update.go,update_test.go}, commands.go, cli_test.go,
specs/update.md, todo/todo.md.

## Run and verify

`make check`. By hand: build with `-ldflags "-X main.version=0.9.0"`
into a directory path containing `/Cellar/lazyai/` and run
`lazyai update --check`.

## Next step

Release with the next batch of changes (specs/release.md).
