package substratekind

// vm_body_test.go — the vm BODY's own behavioural test, moved here with the body
// (opencharly/spec#304 unit 8).
//
// It used to call spec's `defValue(t, "#Vm")` against the shipped schema; the def lives in THIS
// plugin's CUE module now, so the harness composes exactly what the host's gate composes — the
// spec base and this plugin's served unit — and asserts against THAT value. The property is the
// same: a krun guest accepts its own shapes and rejects every libvirt/qemu-only field, naming
// the field it rejects.

import (
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
)

// composedVm looks up #Vm in the composed unit (base & served unit).
func composedVm(t *testing.T) (cue.Value, *cue.Context) {
	t.Helper()
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#Vm"))
	if !def.Exists() {
		t.Fatal("#Vm does not resolve in the composed unit")
	}
	return def, v.Context()
}

// allErrors flattens CUE's error tree. The top-level Error() truncates to
// "(and N more errors)", so a field-named conflict can be invisible unless every
// node is printed.
func allErrors(err error) string {
	var b strings.Builder
	for _, e := range cueerrors.Errors(err) {
		fmt.Fprintf(&b, "%v | path=%v | ", e.Error(), e.Path())
	}
	return b.String()
}

// checker builds an accept/reject predicate over one definition.
func checker(def cue.Value, ctx *cue.Context) func(string) error {
	return func(body string) error {
		unified := def.Unify(ctx.CompileString(body))
		if unified.Err() != nil {
			return unified.Err()
		}
		return unified.Validate(cue.Concrete(false))
	}
}

func TestVmBackendKrunCrossRules(t *testing.T) {
	def, ctx := composedVm(t)
	check := checker(def, ctx)

	const containerDisk = `source: {kind: "container_disk", image: "registry.example/disk@sha256:0000000000000000000000000000000000000000000000000000000000000000"}`

	accepted := []string{
		`{backend: "krun", ` + containerDisk + `}`,
		`{backend: "krun", ` + containerDisk + `, krun: {network: "passt", kernel: "image", nested_virt: true, gpu: "virtio", gpu_flags: 962, ram_headroom_mib: 256}}`,
		`{backend: "krun", source: {kind: "bootc", box: "pod-crabbox-systemd"}}`,
	}
	for _, body := range accepted {
		if err := check(body); err != nil {
			t.Errorf("#Vm rejects an accepted krun shape %s: %v", body, err)
		}
	}

	// Every libvirt/qemu-only field must be rejected WITH backend: krun.
	libvirtOnly := []string{
		`libvirt: {}`,
		`autostart: true`,
		`firmware: "uefi-insecure"`,
		`machine: "q35"`,
		`cloud_init: {hostname: "guest"}`,
	}
	for _, field := range libvirtOnly {
		err := check(`{backend: "krun", ` + field + `, ` + containerDisk + `}`)
		if err == nil {
			t.Errorf("#Vm accepts backend: krun with the libvirt-only %s; a krun guest has no libvirt domain", field)
			continue
		}
		// The rejection must NAME the authored field (or, for autostart, the
		// backend rule it violates), not collapse into an anonymous "no arm
		// matches" union error.
		want := strings.SplitN(field, ":", 2)[0]
		if want == "autostart" {
			// The pre-existing autostart:true ⇒ backend auto|libvirt rule fires
			// first and reports at #Vm.backend; the krun conflict is the same
			// rejection, named at the rule that owns it.
			want = "backend"
		}
		if text := allErrors(err); !strings.Contains(text, want) {
			t.Errorf("the krun rejection of %s does not name %q in its error: %s", field, want, text)
		}
	}

	// A non-OCI source arm is rejected too: krun cannot boot a cloud image URL.
	if err := check(`{backend: "krun", source: {kind: "cloud_image", url: "https://example.com/disk.qcow2"}}`); err == nil {
		t.Error("#Vm accepts backend: krun with a cloud_image source; krun requires an OCI-image source")
	}

	// Positive control: the SAME fields are legal with the libvirt backend, so
	// the rejections above are about the backend, not about the fields.
	control := `{backend: "libvirt", libvirt: {}, autostart: true, firmware: "uefi-insecure", machine: "q35", cloud_init: {hostname: "guest"}, ` + containerDisk + `}`
	if err := check(control); err != nil {
		t.Errorf("#Vm rejects the same fields under backend: libvirt (the krun rejection is not backend-specific): %v", err)
	}
}
