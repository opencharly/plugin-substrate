package substratekind

// selectors.go — the ONE source for the substrate SELECTOR declaration. A
// substrate's #DeployTraits.selectors is spec DATA: the closed vocabulary lives
// in the spec module's schema (deploy.cue, def #DeployTraits), and the loader
// consumes it to reject `engine:` on a vm and `backend:` on a pod without
// switching on the substrate kind word (the boundary law).
//
// This package spells NO selector word. A selector's name is the yaml name of the
// spec.Deploy field that carries it, so deriving it here keeps the declaration in
// step with the contract by construction: a renamed field is a loud init panic, and
// the tests prove the derived word is a member of spec's closed vocabulary and is
// honoured by the resolver that consumes that level. The association with a
// concept is the field name the resolver's own Selection struct carries
// (Engine / Backend / Rootful), never a second vocabulary.

import (
	"reflect"
	"strings"

	"github.com/opencharly/spec/spec"
)

// deploySelectorName returns the authored selector word for one spec.Deploy
// field: its own yaml tag. A field missing from the contract, or carrying no yaml
// name, panics at init rather than silently declaring no selector.
func deploySelectorName(field string) string {
	f, ok := reflect.TypeOf(spec.Deploy{}).FieldByName(field)
	if !ok {
		panic("substrate: spec.Deploy has no field " + field + " — the selector declaration cannot be derived")
	}
	name := strings.Split(f.Tag.Get("yaml"), ",")[0]
	if name == "" || name == "-" {
		panic("substrate: spec.Deploy field " + field + " carries no yaml name — the selector declaration cannot be derived")
	}
	return name
}

// The selector words the sdk resolvers consume, each named after the spec.Deploy
// field that carries it (and therefore after the deploykit.Selection field the
// resolver reads).
var (
	selectorEngine  = deploySelectorName("Engine")
	selectorBackend = deploySelectorName("Backend")
	selectorRootful = deploySelectorName("Rootful")
)
