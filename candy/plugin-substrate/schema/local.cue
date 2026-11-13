// schema/local.cue — the `kind: local` ENTITY body, owned by the plugin that SERVES the kind.
//
// THE KIND BODY LIVES HERE, not in the contract module: a `kind` body belongs with the
// plugin that serves the kind. This plugin's `kind: local` capability declares the gate def
// (`plugin.go`: `InputDef: "#LocalValue"`), so the host resolves `#LocalValue` from that
// DECLARATION and looks it up in the schema composed from the base and every served plugin
// unit — no kernel change and no def-name knowledge in the host.
// opencharly/spec#304 unit 4 moved it here.
//
// WHAT STAYS IN THE CONTRACT, and why: the shared defs this body uses — `#CandyRef`,
// `#InstallOpts`, `#StrVal`, `#Step` — and the `#DeployValue` disjunct are all cited from
// the contract module (`schema.#…`), because a plugin's CUE module can import ONLY the
// contract module. `#DeployValue` is the contract's deploy arm, not a second body: a
// `local:` node whose body has the deploy shape is gated by the SAME def, so the arm is
// cited rather than restated here.
//
// THE IMPORT RESOLVES OFFLINE. spec/schemacompose serves
// `github.com/opencharly/spec/schema` from the spec Go module the binary already links — no
// network, no module cache, no `cue.mod`, no published CUE module — on the serve side
// (sdk.BuildCapabilitiesWithRequires → ServedSchema) AND on the host side, which composes
// the SAME body under the DERIVED module declaration (registerPluginUnitSchema →
// ServedText).
//
// FILE ORDER IS LOAD-BEARING. spec/schemaconcat.ConcatSchema sorts a schema dir's files and
// concatenates them, stripping only the package clause, and a CUE `import` must precede
// every declaration. So the module's single import lives in the file that sorts FIRST:
// `local.cue` sorts before `pod.cue` and `substrate.cue`, and it CARRIES that import now
// that the `kind: local` body has landed. Renaming any of the three without re-checking the
// order makes the served schema uncompilable at load time.

import "github.com/opencharly/spec/schema"

// #Local validates ONE entry of the `local:` map (a target:local candy stack). CLOSED — the
// Go loader decodes with KnownFields(true), so an unknown key is an error. Shared defs
// REFERENCED, not redefined (R3).
#Local: {
	// Ordered candy stack applied to the host. Required (empty list permitted — a staged
	// name-reservation stub; the loader warns, not errors).
	candy!: [...schema.#CandyRef]
	install_opts?: schema.#InstallOpts @go(InstallOpts,optional=nillable)
	env?: {PATH?: _|_, [string]: schema.#StrVal} @go(Env,type=map[string]string)
	description?: string & !=""
	plan?: [...schema.#Step]
}

// #LocalValue — the HOST-SIDE value gate def for the `local` kind (the #Candy /
// #<Kind>Value analogue). @go(-): the plugin keeps no typed Go body for it — the one
// consumer, resolve.go's OpResolve leg, decodes the authored template straight into the
// contract's #ResolvedLocal envelope. Its NAME is the contract the `kind` capability
// declares (plugin.go's InputDef), which is how the HOST resolves the gate for this kind
// without carrying any knowledge of it.
//
// The #DeployValue disjunct is the contract's deploy arm, NOT a second body (see above).
#LocalValue: (#Local | schema.#DeployValue) @go(-)
