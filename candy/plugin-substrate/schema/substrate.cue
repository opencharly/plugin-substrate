// plugin-substrate's OWN self-contained CUE schema — the SINGLE SOURCE for this
// plugin's declaration surface, used two ways exactly like every other plugin's
// schema (there is no schema-less plugin):
//
//  1. GENERATE the Go params — `cue exp gengotypes` → ../params/cue_types_gen.go.
//  2. SERVE over Describe — the host splices `base ++ plugin` at the load gate
//     (registerPluginUnitSchema), so the plugin's declarations travel WITH it and
//     a self-contained schema that will not splice is a LOUD load failure.
//
// The substrate kinds' values are RICH + core-referencing (#Vm/#Deploy/…) and are
// validated HOST-SIDE against the kept #<Kind>Value core def, so this schema does
// NOT define an input def for them (each capability declares InputDef:""). It
// DOCUMENTS the provider's declaration surface — the kind/command/verb words and
// the deploy traits they carry — and satisfies the uniform non-empty-schema
// contract. SELF-CONTAINED: it references no base def, so it compiles STANDALONE
// (the property `cue exp gengotypes` needs and the property that lets the SDK
// compile it serve-side).
#SubstratePlugin: {
	// The structural deploy kind words this provider serves.
	kinds: [...string]

	// The command words this provider serves (reap-orphans).
	commands: [...string]

	// The internal-only verb words this provider serves (status-fanout).
	verbs: [...string]

	// What this provider does, in one line (the public-docs surface).
	contract: string & !=""

	// The configuration surface: env var names the collectors read.
	config?: [string]: string
}
