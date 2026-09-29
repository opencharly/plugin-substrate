# plugin-substrate

Deploy-substrate structural kinds for OpenCharly — the `pod`, `vm`, `kubernetes`,
`local`, `android`, `kubevirt`, and `kindcluster` kinds.

`plugin-substrate` owns the substrate structural deploy kinds. Each is **both** a
standalone **template** (a bare `vm:`/`pod:` block → the typed template map
`uf.Pod`/`uf.VM`/…, the primary VM authoring form) **and** a **deploy**
(`from:`/`image:` cross-ref or resource members → `uf.Deploy`). The host
pre-decodes the canonical node via the core loader and threads it in `op.Env`;
this plugin's `OpLoad` echoes it and the host folds the echo. It is **compiled-in**
(in the embedded `compiled_plugins:`), like the tier-1 kinds, because these are
core deploy primitives every box/submodule must always resolve; it also serves
out-of-process via `cmd/serve`.

The plugin also serves the **`OpStatusCollect`** collector — the cleanly-movable
`charly status` collectors (pod live + local install-ledger + the probes, plus the
vm/kubernetes/kubevirt/android collectors) reached by the host's status fan-out.

## What it provides

| Capability | Surface |
|---|---|
| `kind:pod` | the `pod:` structural kind (template + deploy fold) |
| `kind:vm` | the `vm:` structural kind |
| `kind:kubernetes` | the `kubernetes:` structural kind |
| `kind:local` | the `local:` structural kind |
| `kind:android` | the `android:` structural kind |
| `kind:kubevirt` | the `kubevirt:` structural kind (a KubeVirt VM on a cluster) |
| `kind:kindcluster` | the `kindcluster:` structural kind |
| `OpStatusCollect` | the status collectors reached by the `charly status` fan-out |

## How to use it

It is composed by every box/check bed that authors a substrate node; no authored
configuration is required. It is transparent to users — you write `pod:`/`vm:`/
`local:`/… nodes as usual.

## R10 witness bed

The repo's root `charly.yml` declares `eval-vm` and the `disposable: true`
`check-substrate` bed: a bare `local:` template plus a nested `local:` deploy
member, both decoded through this plugin inside the disposable VM guest, with the
marker check asserting `/etc/check-substrate-marker` in the guest.

## Layout

- `candy/plugin-substrate/` — the plugin module: `plugin.go` (the kind providers +
  `NewMeta()`), `resolve.go` (the structural decode), `status_*.go` (the
  collectors), `validate_*.go`, `schema/substrate.cue`, `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy` + the witness bed).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-internals:plugin` — the plugin/provider model and the
  kind-decode seam this provider implements. This candy carries no `skill:`
  entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-core:charly-status` — the status surface whose fan-out reaches the
  collectors here.
- `/charly-internals:install-plan` — the deploy IR the substrate kinds fold into.
