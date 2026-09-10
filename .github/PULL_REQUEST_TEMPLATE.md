## What changed

<!-- One paragraph. What does this PR do, and why is it needed? -->

## Why this approach

<!-- Alternatives considered, and why this one won. Delete for trivial changes. -->

## How it was verified

<!-- Commands you actually ran, and what they reported. -->

```
scripts/check.sh
```

<!-- gofmt -> go vet ./... -> go test ./... -> golangci-lint run -> go build ./... -->

## Scope

- [ ] Adding or changing a language plugin (`internal/lang`)
- [ ] Changing the plugin contract (`internal/plugin`) — see `docs/adr/`
- [ ] Changing the CLI surface (`internal/cli`)
- [ ] Documentation only

## Definition of done

- [ ] The validation flow passes end to end, in order:
  - [ ] 1. `gofmt` reports nothing
  - [ ] 2. `go vet ./...` is clean
  - [ ] 3. `go test ./...` passes, including new tests for the changed behaviour
  - [ ] 4. `golangci-lint run` is clean
  - [ ] 5. `go build ./...` succeeds
- [ ] `go test -race ./...` passes, or CI will cover it
- [ ] `go mod tidy` produces no diff
- [ ] Exported identifiers have doc comments
- [ ] Adding a language required no change to core packages, or the ADR explains why it did
- [ ] Docs updated (`README.md`, `CLAUDE.md`, `docs/`) if behaviour changed
- [ ] No secrets, tokens or personal paths in the diff

## Notes for reviewers

<!-- Anything worth a second pair of eyes. -->
