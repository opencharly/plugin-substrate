// plugin-substrate's OWN self-contained CUE schema — the SINGLE SOURCE for this plugin's
// served declaration surface (there is no schema-less plugin: every plugin ships a
// non-empty schema over Describe).
//
// SELF-CONTAINED and PACKAGE-LESS: it references no base def and carries no package
// clause, so it compiles STANDALONE — the property the SDK's serve-side compile needs
// and the property that lets the host splice `base ++ plugin` at the load gate
// (registerPluginUnitSchema); a self-contained schema that will not splice is a LOUD
// load failure.
//
// NO GO CONSUMER: the plugin declares no typed `plugin_input` (its authored input is
// its pass-through CLI grammar), so this schema generates NO `params` package and has
// NO `cue exp gengotypes` artifact — it is the SERVED documentation/config surface,
// not a code-generation source.
//
// It DOCUMENTS the substrate provider's kind/command/verb words and the deploy traits they carry. The rich substrate VALUES are validated HOST-SIDE against the kept `#<Kind>Value` defs (each capability declares `InputDef:""`), not decoded here.
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
