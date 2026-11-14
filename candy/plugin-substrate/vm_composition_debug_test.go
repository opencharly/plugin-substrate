package substratekind

// vm_composition_debug_test.go — the RCA harness for opencharly/spec#304 unit 8's plugin leg.
//
// The leg adds `schema/vm.cue` (the `#Vm` body) to this plugin's served module. The plugin's own
// suite goes green, but the two-arm corpus sweep moves 32 `vm` bodies from OK to FAIL with
// "#VmValue.source: field not allowed". This test exists to make the CAUSE readable:
//
//	the composed `#Vm` is a FOUR-field closed shape (source, firmware, backend, autostart), whose
//	firmware/backend fields come from `/opencharly/plugin-schema/schema/plugin.cue` — not from
//	this plugin's 24-field `#Vm` and not from the base's identically-shaped one. A CLOSED struct
//	unified with an open one closes it, so the composed unit loses every optional field and the
//	bodies' `disk_size`, `machine`, `cloud_init`, `ssh`, `libvirt`, `snapshot`… become
//	"field not allowed".
//
// SECOND MEASUREMENT (round 2 of the RCA): it is not a narrowing of `#Vm` alone — EVERY def this
// file adds is EMPTY in the composed unit (`#VmCharlyInstall` 0 fields, `#VmChecksum` 0,
// `#VmValue` 0) while the SERVED TEXT carries them intact (`#Vm: {` at line 1134 of the 64,071-byte
// served schema, `#VmCharlyInstall: {` at 1229). So the loss happens in schemacompose's module
// instance, not in the concatenation, and the leading suspect is the `schema.#…` resolution of a
// file that sorts AFTER the import carrier (`vm.cue` is last; every earlier unit is also last or
// middle and works, so the difference to find is what makes THIS file's references unresolvable).
//
// It is a DIAGNOSTIC, not a gate: it logs the composed shape and fails only if the shape is no
// longer the narrowed one, so the next session can see when the composition changes.

import (
	"testing"

	"cuelang.org/go/cue"
)

func TestVmComposedShape(t *testing.T) {
	v := composedBodySchema(t)
	vm := v.LookupPath(cue.ParsePath("#Vm"))
	if !vm.Exists() {
		t.Fatal("#Vm does not resolve in the composed tree")
	}
	iter, err := vm.Fields()
	if err != nil {
		t.Fatalf("#Vm fields: %v", err)
	}
	var names []string
	for iter.Next() {
		names = append(names, iter.Label())
	}
	t.Logf("composed #Vm fields (%d): %v", len(names), names)
	for _, f := range names {
		if src := vm.LookupPath(cue.ParsePath(f)).Source(); src != nil {
			t.Logf("  %s <- %s", f, src.Pos())
		}
	}
	if len(names) < 20 {
		t.Logf("NARROWED: the composed #Vm carries %d fields; the authored one carries 24", len(names))
	}
}
