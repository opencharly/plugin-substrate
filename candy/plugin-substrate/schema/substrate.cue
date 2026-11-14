// plugin-substrate's OWN CUE schema (with schema/pod.cue) — the SINGLE SOURCE for this
// plugin's served declaration surface (there is no schema-less plugin: every plugin ships a
// non-empty schema over Describe).
//
// ONE CUE MODULE PACKAGE, FIVE FILES. This file carries the declaration surface and
// references no base def; `kindcluster.cue` carries the `kind: kindcluster` ENTITY body and
// the module's single contract import, `kubernetes.cue` the `kind: kubernetes` body (with its
// own policy sub-defs and vocabulary tables), and `local.cue` / `pod.cue` their bodies.
// The module composes as ONE package — `schemaconcat.ConcatSchema` concatenates a schema
// dir's files in sorted order and strips only the package clause, so the single `import`
// lives in the file that sorts FIRST (`kindcluster.cue`, before the other four). It is composed by
// the SAME code on both sides of the wire — spec/schemacompose on the serve side
// (sdk.BuildCapabilitiesWithRequires → ServedSchema) and on the host side
// (registerPluginUnitSchema → ServedText, which unifies the unit onto the base); a unit that
// will not compose alone or will not splice is a LOUD load failure, never a silent skip.
//
// NO GO CONSUMER: the plugin declares no typed `plugin_input` (its authored input is
// its pass-through CLI grammar), so this schema generates NO `params` package and has
// NO `cue exp gengotypes` artifact — it is the SERVED documentation/config surface,
// not a code-generation source. The pod body's one consumer (resolve.go) decodes the
// authored bytes straight into the contract's `#ResolvedPod` envelope, so it needs no
// generated type either.
//
// It DOCUMENTS the substrate provider's kind/command/verb words and the deploy traits they carry. The rich substrate VALUES are validated HOST-SIDE against the kept `#<Kind>Value` defs. Each capability declares `InputDef:""` — the base still declares its def — EXCEPT `pod`, whose body lives here and whose capability declares `InputDef:"#PodValue"`.
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
