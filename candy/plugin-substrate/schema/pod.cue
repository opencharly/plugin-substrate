// schema/pod.cue — the `kind: pod` ENTITY body, owned by the plugin that SERVES the kind.
//
// THE KIND BODY LIVES HERE, not in the contract module: a `kind` body belongs with the
// plugin that serves the kind. This plugin's `kind: pod` capability declares the gate def
// (`plugin.go`: `InputDef: "#PodValue"`), so the host resolves `#PodValue` from that
// DECLARATION and looks it up in the schema composed from the base and every served plugin
// unit — no kernel change and no def-name knowledge in the host.
// opencharly/spec#304 unit 3 moved it here.
//
// WHAT STAYS IN THE CONTRACT, and why: `#PodSidecar` is cited by the contract's own
// `#ResolvedPod` (spec/schema/substrate_template.cue) — the resolve envelope the kernel
// reads — so the def is referenced here as `schema.#PodSidecar` and stays where it is. A
// plugin's CUE module can import ONLY the contract module, which is exactly why that
// reference settles the question rather than taste.
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
// `firecracker.cue` (which carries the `kind: firecracker` body and with it the import),
// before every other file. Renaming any of them without re-checking that order makes the
// served schema uncompilable at load time.

// #Pod validates ONE entry of the `pod:` map (PodSpec). Most projects declare no kind:pod
// entries (the real corpus is all `pod: {}`); the schema is modeled directly from the
// struct. CLOSED — every PodSpec field is modeled, so an unknown key is a typo. Shared defs
// REFERENCED, not redefined (R3): schema.#Step / schema.#CandyRef (the contract's),
// schema.#DeploySecret from deploy.cue, schema.#Sidecar from sidecar.cue.
#Pod: {
	// References a kind:box (bare lowercase-hyphenated name or remote ref).
	// Optional: the Go field has no non-empty validator.
	box?: schema.#CandyRef
	// Each entry is a SidecarConfig wrapper: a single-key map of named
	// sidecar templates (PodSpec.Sidecar []SidecarConfig).
	sidecar?: [...schema.#PodSidecar]
	secret?: [...schema.#DeploySecret]
	env_default?: {[string]: schema.#StrVal} @go(EnvDefaults,type=map[string]string)
	plan?: [...schema.#Step]
}

// #PodValue — the HOST-SIDE value gate def for the `pod` kind (the #Candy / #<Kind>Value
// analogue). @go(-): the plugin keeps no typed Go body for it — the one consumer,
// resolve.go's OpResolve leg, decodes the authored template straight into the contract's
// #ResolvedPod envelope. Its NAME is the contract the `kind` capability declares
// (plugin.go's InputDef), which is how the HOST resolves the gate for this kind without
// carrying any knowledge of it.
//
// The #DeployValue disjunct is the contract's deploy arm, NOT a second body: a `pod:` node
// whose body has the deploy shape (from:/image:/members) is gated by the SAME def, so the
// arm is cited from the contract rather than restated here.
#PodValue: (#Pod | schema.#DeployValue) @go(-)
