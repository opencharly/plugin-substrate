package substratekind

// firecracker_body_test.go — the firecracker BODY's own validation tests, moved here with the
// body (opencharly/spec#304 unit 7).
//
// They used to compile spec's shipped schema and look up #FirecrackerValue; the def lives in THIS
// plugin's CUE module now, so the harness composes exactly what the host's gate composes — the
// spec base and this plugin's served unit — and asserts against THAT value. Every acceptance check
// asserts COMPLETENESS (cue.Concrete(true)): plain Validate() treats a missing required field as
// incomplete rather than erroneous and would report success, and a positive gate that cannot fail
// is not a gate.

import (
	"testing"

	"cuelang.org/go/cue"
)

// validFirecracker bodies are unified against the def and checked with
// cue.Concrete(true) — NOT the plain Validate(). Plain Validate() treats a MISSING
// REQUIRED FIELD as incomplete rather than erroneous, so it reports success for a
// body that omits a required field; measured: with `image` required, a host-only body
// passes plain Validate() and fails Validate(cue.Concrete(true)). A positive gate that
// cannot fail is not a gate (R7), so every acceptance check below asserts completeness.
func unifyValid(t *testing.T, v cue.Value, def cue.Value, body string) error {
	t.Helper()

	// compile in the DEF'S OWN context: a cue.Value is only valid inside the context that built
	// it, and a fresh cuecontext.New() here would silently produce an empty-disjunction error
	// (the lesson the kubevirt tests paid for).
	return def.Unify(v.Context().CompileString(body)).Validate(cue.Concrete(true))
}

// TestFirecrackerValue_TemplateAccepted passes a realistic kind:firecracker template
// (the host surface + a microVM instance + a runner pool) through the host-side value
// gate def #FirecrackerValue — now read from the COMPOSED unit (base & this plugin's served
// schema), the same value the host's gate builds — proving the authored surface validates.
// It asserts COMPLETENESS (cue.Concrete(true)), so a body missing a required field cannot pass.
func TestFirecrackerValue_TemplateAccepted(t *testing.T) {
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#FirecrackerValue"))
	if !def.Exists() {
		t.Fatal("#FirecrackerValue is not defined in the shipped schema")
	}
	body := `{
	vmm: {version: "v1.17.0", sha256: "06094a1108ae9e82aa4c23a775aa92758f53f1175d422270d9d6162cb9ade558", jailer: {enabled: false}}
	kernel: {from: "ghcr.io/opencharly/firecracker-kernel:v0.1", cmdline: "console=ttyS0 reboot=k panic=1 pci=off nomodules rw"}
	network: {
		mode: "tap"
		tap: {name: "fc-tap{n}", subnet: "172.16.0.0/16", prefix: 30, host_offset: 1}
		egress_iface: "auto"
		forwarding: {ip_forward: true, policy: "append-accept"}
		resolver: ["1.1.1.1", "8.8.8.8"]
	}
	containerd: {
		snapshotter: "blockfile"
		manage: "merge"
		devmapper: {pool: "containerd-thinpool", base_image_size: "30GB", device: "/dev/nvme1n1", profile: {thin_pool_autoextend_threshold: 80}}
	}
	fireactions: {
		version: "v2.0.8"
		install: {from_source: true}
		bind_address: "127.0.0.1:8080"
		github: {app_id: 123456, app_private_key_secret: "charly/fireactions/app-key"}
		pools: [{
			name: "opencharly-2vcpu-4gb"
			replicas: 2
			runner: {image: "ghcr.io/opencharly/fireactions-runner:v0.1", organization: "opencharly", group_id: 1, labels: ["self-hosted", "fireactions"]}
			firecracker: {machine_config: {vcpu_count: 2, mem_size_mib: 4096}, metadata: {pool: "opencharly-2vcpu-4gb"}}
		}]
	}
	image: {from: "ghcr.io/opencharly/fireactions-runner:v0.1", format: "ext4", disk_mib: 6144, resize: true}
	machine: {
		vcpu_count: 2
		mem_size_mib: 4096
		drives: [{drive_id: "work", path: "/var/lib/firecracker/work.ext4", io_engine: "Async"}]
		rate_limits: {network_out: {bandwidth: {size: 26214400, refill_time: 1000, one_time_burst: 0}}}
	}
	mmds: {version: "V2", metadata: {charly: {deploy: "runner-vm"}}}
	cloud_init: {hostname: "fc-1", package: ["sudo"]}
	vsock: {guest_cid: 3}
	console: {serial_out_path: "/var/log/fc-1.serial", log_level: "Info"}
}`
	if err := unifyValid(t, v, def, body); err != nil {
		t.Fatalf("a valid kind:firecracker template rejected by #FirecrackerValue: %v", err)
	}
}

// TestFirecrackerValue_HostOnlyTemplateAccepted gates the other authored shape: a
// template that names the kernel and the host's capabilities but NO image is legal,
// because that is the host profile the instances inherit from (a deploy names such a
// template via `from:` and brings its own image). The deployed shape's requirement is
// the plugin's OpValidate, so this gate is what keeps the host form authorable: making
// `image` required again fails it.
func TestFirecrackerValue_HostOnlyTemplateAccepted(t *testing.T) {
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#FirecrackerValue"))
	if !def.Exists() {
		t.Fatal("#FirecrackerValue is not defined in the shipped schema")
	}
	body := `{
	kernel: {from: "ghcr.io/opencharly/firecracker-kernel:v0.1", cmdline: "console=ttyS0 rw"}
	vmm: {version: "v1.17.0"}
	network: {mode: "tap", tap: {subnet: "172.16.0.0/16", prefix: 30}}
	containerd: {snapshotter: "blockfile"}
	fireactions: {
		version: "v2.0.8"
		pools: [{name: "opencharly-2vcpu-4gb", runner: {image: "ghcr.io/opencharly/fireactions-runner:v0.1", labels: ["self-hosted"]}}]
	}
}`
	if err := unifyValid(t, v, def, body); err != nil {
		t.Fatalf("a host-only kind:firecracker template (no image) rejected by #FirecrackerValue: %v", err)
	}
}

// TestFirecrackerValue_RequiredScalarEnforced gates that the def's required non-empty
// scalars really are required: a pool whose runner omits `image`, or sets it to the
// empty string, must fail. Measured behaviour, not an assumption: `Concrete(true)` on
// the unified value reports `fireactions.pools.0.runner.image: incomplete value !=""`
// and `invalid value "" (out of bound !="")` respectively.
//
// Note for the next reader: a missing field whose value is a plain struct REFERENCE
// (`kernel: #FirecrackerKernel`) is NOT flagged by that call in this unification mode,
// which is why this gate targets a required scalar rather than an absent struct field —
// a gate that cannot fail is not a gate (R7).
func TestFirecrackerValue_RequiredScalarEnforced(t *testing.T) {
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#FirecrackerValue"))
	for _, tc := range []struct{ name, body string }{
		{"runner without an image", `{kernel: {from: "k"}, fireactions: {pools: [{name: "p", runner: {labels: ["a"]}}]}}`},
		{"runner with an empty image", `{kernel: {from: "k"}, fireactions: {pools: [{name: "p", runner: {image: "", labels: ["a"]}}]}}`},
	} {
		if err := unifyValid(t, v, def, tc.body); err == nil {
			t.Fatalf("%s must fail the gate: a pool's runner image is required and must be non-empty", tc.name)
		}
	}
}

// TestFirecrackerValue_UnknownKeyRejected proves the new def is CLOSED (a typo is an
// error, not a silent drop) — the closedness the host value gate relies on.
func TestFirecrackerValue_UnknownKeyRejected(t *testing.T) {
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#FirecrackerValue"))
	body := `{kernel: {path: "/boot/vmlinux"}, image: {from: "x"}, bogus_key: 1}`
	if err := def.Unify(v.Context().CompileString(body)).Validate(); err == nil {
		t.Fatal("a kind:firecracker template with an unknown key must fail the closed gate")
	}
}

// TestFirecrackerValue_SnapshotterEnumRejected proves the enum'd sub-shapes carry
// their own strictness: a snapshotter outside the two that can back a root drive is
// refused by the schema, not left to the plugin — and the two legal values still
// pass, so the gate is the enum and not a blanket rejection.
func TestFirecrackerValue_SnapshotterEnumRejected(t *testing.T) {
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#FirecrackerValue"))

	for _, ok := range []string{"devmapper", "blockfile"} {
		body := `{kernel: {path: "/boot/vmlinux"}, image: {from: "x"}, containerd: {snapshotter: "` + ok + `"}}`
		if err := unifyValid(t, v, def, body); err != nil {
			t.Fatalf("containerd.snapshotter %q must be accepted: %v", ok, err)
		}
	}

	for _, bad := range []string{"overlayfs", "native"} {
		body := `{kernel: {path: "/boot/vmlinux"}, image: {from: "x"}, containerd: {snapshotter: "` + bad + `"}}`
		if err := def.Unify(v.Context().CompileString(body)).Validate(); err == nil {
			t.Fatalf("containerd.snapshotter %q must fail: its mount is a directory, which cannot back a microVM root drive", bad)
		}
	}
}
