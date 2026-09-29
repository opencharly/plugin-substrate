# AGENTS.md — plugin-substrate

Standalone plugin repo for the deploy-substrate structural kinds (`kind:pod`,
`kind:vm`, `kind:kubernetes`, `kind:local`, `kind:android`, `kind:kubevirt`,
`kind:kindcluster`) and the `OpStatusCollect` collectors. The plugin is a Go
module at `candy/plugin-substrate/` (module path
`github.com/opencharly/plugin-substrate/candy/plugin-substrate`); the root
`charly.yml` declares `discover: candy` **and** the `check-substrate` R10 witness
bed.

Canonical files:

- `candy/plugin-substrate/charly.yml` — the `plugin-substrate:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-substrate/plugin.go` — the kind providers + `NewMeta()`.
- `candy/plugin-substrate/resolve.go` — the structural template/deploy decode.
- `candy/plugin-substrate/status_*.go` — the status collectors
  (`OpStatusCollect`).
- `candy/plugin-substrate/schema/substrate.cue` — the self-contained schema.
- `charly.yml` — the project manifest + the `check-substrate` bed.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model, the kind-decode seam, the per-plugin
  CUE-schema contract, placement. Load before touching the provider or schema.
- `/charly-core:charly-status` — the status surface whose fan-out reaches the
  collectors here.
- `/charly-internals:install-plan` — the deploy IR the substrate kinds fold into.
- `/charly-check:check` — the disposable bed / R10 run sequence.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-substrate/` — compile the plugin module.
- `go test ./...` in `candy/plugin-substrate/` — the plugin's Go tests (the
  `status_*_test.go` collectors, `resolve_test.go`, `load_pod_substrate_test.go`,
  the relocated golden `status_test.go`).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema, the witness bed).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- R10 witness: run the `disposable: true` `check-substrate` bed via
  `charly check run` (see `/charly-check:check`).

## Modify this repo

- Edit the `plugin-substrate:` candy entity, the Go source, and
  `schema/substrate.cue` **together** — the schema is the single source for the
  generated types.
- The host pre-decodes the canonical node and this plugin ECHOES it; keep the
  decode source of truth in the host loader, not here.
- The plugin is **compiled-in** (core deploy primitives); it also serves
  out-of-process via `cmd/serve`.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
