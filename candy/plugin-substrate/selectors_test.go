package substratekind

// selectors_test.go — the drift control for the substrate SELECTOR declaration.
//
// Two independent claims, each read from the producer rather than restated:
//
//  1. every selector a trait declares is a member of spec's CLOSED vocabulary
//     (#DeployTraits.selectors in the spec module's embedded schema); and
//  2. every selector a trait declares is actually CONSUMED by the sdk resolver
//     that owns that level — proven by asking the resolver for an answer that
//     changes when the level is authored.
//
// A trait entry that declares a word spec does not accept, or a word no resolver
// reads, fails here instead of shipping a declaration nothing honours.

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/opencharly/sdk/deploykit"
	"github.com/opencharly/spec/schema"
	"github.com/opencharly/spec/schemaconcat"
	"github.com/opencharly/spec/spec"
)

// specSelectorVocabulary reads the closed selector vocabulary from the spec
// module's embedded schema — the SAME #DeployTraits def the loader validates
// against, never a copy of its words.
func specSelectorVocabulary(t *testing.T) map[string]bool {
	t.Helper()
	body, files, err := schemaconcat.ConcatSchema(schema.FS, ".", nil)
	if err != nil {
		t.Fatalf("read the spec schema: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("the spec schema is empty; a vocabulary read from it would be vacuous")
	}
	root := cuecontext.New().CompileString(body)
	if err := root.Err(); err != nil {
		t.Fatalf("compile the spec schema: %v", err)
	}
	traits := root.LookupPath(cue.ParsePath("#DeployTraits"))
	if err := traits.Err(); err != nil {
		t.Fatalf("#DeployTraits is absent from the spec schema: %v", err)
	}
	fields, err := traits.Fields(cue.Optional(true))
	if err != nil {
		t.Fatalf("#DeployTraits is not a struct: %v", err)
	}
	vocabulary := map[string]bool{}
	found := false
	for fields.Next() {
		// #DeployTraits.selectors is the closed selector vocabulary; every trait
		// field is optional, so the iterator must include optional fields.
		if fields.Selector().Unquoted() != "selectors" {
			continue
		}
		found = true
		elem := fields.Value().LookupPath(cue.MakePath(cue.AnyIndex))
		if err := elem.Err(); err != nil {
			t.Fatalf("#DeployTraits.selectors is not a list: %v", err)
		}
		op, parts := elem.Expr()
		if op != cue.OrOp || len(parts) == 0 {
			t.Fatalf("#DeployTraits.selectors element is not a closed disjunction (op=%v, parts=%d)", op, len(parts))
		}
		for _, part := range parts {
			word, err := part.String()
			if err != nil {
				t.Fatalf("read a selector word: %v", err)
			}
			vocabulary[word] = true
		}
	}
	if !found {
		t.Fatal("#DeployTraits.selectors is absent from the spec schema")
	}
	if len(vocabulary) == 0 {
		t.Fatal("read zero selector words; the instrument is pointed at the wrong object")
	}
	return vocabulary
}

// resolverConsumedSelectors asks each resolver whether the level it owns changes
// its answer. A word is in the result only when authoring that level produced a
// DIFFERENT answer, so a resolver that ignores a level cannot pass.
func resolverConsumedSelectors(t *testing.T) map[string]bool {
	t.Helper()
	if len(spec.EngineNames) == 0 || len(spec.VmBackends) == 0 || len(spec.RootfulModes) == 0 {
		t.Fatal("a spec vocabulary is empty; the consumption probe would be vacuous")
	}
	engineWord := deploySelectorName("Engine")
	backendWord := deploySelectorName("Backend")
	rootfulWord := deploySelectorName("Rootful")
	consumed := map[string]bool{}

	// Engine: a deploy-level engine word is the answer ResolvePodEngine returns.
	wantEngine := spec.EngineName(spec.EngineNames[0])
	if got, err := deploykit.ResolvePodEngine("", deploykit.Selection{Engine: wantEngine}, nil); err != nil {
		t.Fatalf("ResolvePodEngine at the deploy level: %v", err)
	} else if got == wantEngine {
		consumed[engineWord] = true
	}

	// Backend: a deploy-level backend word is the answer ResolveVmBackend returns.
	wantBackend := spec.VmBackend(spec.VmBackends[0])
	if got, err := deploykit.ResolveVmBackend("", deploykit.Selection{Backend: wantBackend}, nil); err != nil {
		t.Fatalf("ResolveVmBackend at the deploy level: %v", err)
	} else if got == wantBackend {
		consumed[backendWord] = true
	}

	// Rootful: at least one concrete rootful mode must change the resolved
	// target's rootful fact. The mode is taken from spec's own vocabulary, so no
	// word is spelled here.
	base, err := deploykit.ResolveContainerdTarget(wantEngine, deploykit.Selection{}, nil)
	if err != nil {
		t.Fatalf("ResolveContainerdTarget baseline: %v", err)
	}
	for _, mode := range spec.RootfulModes {
		got, err := deploykit.ResolveContainerdTarget(wantEngine, deploykit.Selection{Rootful: spec.RootfulMode(mode)}, nil)
		if err == nil && got.Rootful != base.Rootful {
			consumed[rootfulWord] = true
			break
		}
	}
	return consumed
}

// TestSubstrateSelectorsAreDeclaredFromSpec proves every declared selector is in
// spec's closed vocabulary and is consumed by its resolver — the two claims the
// declaration stands on.
func TestSubstrateSelectorsAreDeclaredFromSpec(t *testing.T) {
	vocabulary := specSelectorVocabulary(t)
	consumed := resolverConsumedSelectors(t)

	for word, traits := range substrateTraits {
		if traits == nil {
			t.Fatalf("substrateTraits[%q] is nil", word)
		}
		seen := map[string]bool{}
		for _, selector := range traits.Selectors {
			if seen[selector] {
				t.Errorf("substrateTraits[%q] declares %q twice", word, selector)
			}
			seen[selector] = true
			if !vocabulary[selector] {
				t.Errorf("substrateTraits[%q] declares selector %q, which spec's #DeployTraits.selectors does not accept (vocabulary %v)", word, selector, vocabulary)
			}
			if !consumed[selector] {
				t.Errorf("substrateTraits[%q] declares selector %q, but no sdk resolver consumes that level (consumed %v)", word, selector, consumed)
			}
		}
	}
}

// TestSubstrateSelectorDeclarations pins WHICH substrate accepts which level —
// derived words, never spelled strings. pod is the only engine substrate (engine +
// rootful); vm is the only backend substrate (backend).
func TestSubstrateSelectorDeclarations(t *testing.T) {
	for _, tc := range []struct {
		word string
		want []string
	}{
		{"pod", []string{selectorEngine, selectorRootful}},
		{"vm", []string{selectorBackend}},
	} {
		traits, ok := substrateTraits[tc.word]
		if !ok {
			t.Fatalf("substrateTraits has no %q row", tc.word)
		}
		if len(traits.Selectors) != len(tc.want) {
			t.Errorf("substrateTraits[%q].Selectors = %v, want %v", tc.word, traits.Selectors, tc.want)
			continue
		}
		for i := range tc.want {
			if traits.Selectors[i] != tc.want[i] {
				t.Errorf("substrateTraits[%q].Selectors = %v, want %v", tc.word, traits.Selectors, tc.want)
				break
			}
		}
	}
	for _, word := range []string{"kubernetes", "local", "android", "kubevirt", "kindcluster", "firecracker"} {
		if traits := substrateTraits[word]; traits != nil && len(traits.Selectors) != 0 {
			t.Errorf("substrateTraits[%q].Selectors = %v, want none (the canonical table names selectors only for pod and vm)", word, traits.Selectors)
		}
	}
}
