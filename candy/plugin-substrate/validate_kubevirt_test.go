package substratekind

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencharly/spec/spec"
)

// TestValidateKubevirtDeep covers the XOR rules the closedness-only host gate
// cannot express: the source-arm kind must be known, a container_disk must not
// carry a storage_class, each gpus[] entry must set exactly one of
// resource_name/device_name, and instancetype is exclusive with an explicit cpu.
func TestValidateKubevirtDeep(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr int // number of diagnostics expected
	}{
		{
			name:    "valid container_disk",
			body:    `{"source":{"kind":"container_disk","image":"x"},"gpus":[{"resource_name":"nvidia.com/gpu"}],"instancetype":"u1.medium"}`,
			wantErr: 0,
		},
		{
			name:    "unknown source kind",
			body:    `{"source":{"kind":"bogus"}}`,
			wantErr: 1,
		},
		{
			name:    "empty source kind",
			body:    `{"source":{}}`,
			wantErr: 1,
		},
		{
			name:    "container_disk with storage_class",
			body:    `{"source":{"kind":"container_disk","storage_class":"fast"}}`,
			wantErr: 1,
		},
		{
			name:    "gpu with both fields",
			body:    `{"source":{"kind":"pvc","pvc":"p"},"gpus":[{"resource_name":"a","device_name":"b"}]}`,
			wantErr: 1,
		},
		{
			name:    "gpu with neither field",
			body:    `{"source":{"kind":"pvc","pvc":"p"},"gpus":[{}]}`,
			wantErr: 1,
		},
		{
			name:    "instancetype with explicit cpu",
			body:    `{"source":{"kind":"pvc","pvc":"p"},"instancetype":"u1.medium","cpu":{"cores":2}}`,
			wantErr: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags, err := validateKubevirtDeep(json.RawMessage(tc.body))
			if err != nil {
				t.Fatalf("validateKubevirtDeep: %v", err)
			}
			if len(diags.Items) != tc.wantErr {
				t.Fatalf("got %d diagnostics, want %d: %+v", len(diags.Items), tc.wantErr, diags.Items)
			}
		})
	}
}

// TestStatusCollect_KubevirtDispatch proves the OpStatusCollect word switch routes
// kubevirt to its collector (an unsupported word errors; kubevirt must not).
func TestStatusCollect_KubevirtDispatch(t *testing.T) {
	// With no reverse-channel executor the collector errors on project fetch, but the
	// DISPATCH must reach it (not the "unsupported word" branch). We assert the error
	// message is NOT the unsupported-word one.
	_, err := statusCollect(t.Context(), "kubevirt", []byte(`{}`))
	if err != nil && strings.Contains(err.Error(), "unsupported word") {
		t.Fatalf("kubevirt must be a supported status word, got: %v", err)
	}
}

// TestEphemeralUnderlyingResourceAlive_Kubevirt proves the kubevirt reap probe is
// CONSERVATIVE when it cannot probe (kubectl absent → assume alive, never reap a
// possibly-live VM). It is a pure LookPath guard, so it is deterministic.
func TestEphemeralUnderlyingResourceAlive_Kubevirt(t *testing.T) {
	// Force kubectl to be unfindable for this test.
	t.Setenv("PATH", t.TempDir())
	node := spec.Deploy{Target: "kubevirt", KubeVirtState: &spec.KubeVirtDeployState{VMName: "kv", Namespace: "vms"}}
	if !ephemeralUnderlyingResourceAlive(t.Context(), nil, "kv", node) {
		t.Fatal("kubevirt reap probe must be conservative (alive) when kubectl is absent")
	}
}

// TestEphemeralUnderlyingResourceAlive_KubevirtPresent PROVES the present branch:
// with a fake `kubectl` on PATH it runs `kubectl get vm <vm> -n <ns>` and returns
// alive == (exit 0). This test FAILS if the `case "kubevirt":` arm is removed (the
// fallthrough returns true regardless of the fake's exit), so it gates the change.
func TestEphemeralUnderlyingResourceAlive_KubevirtPresent(t *testing.T) {
	dir := t.TempDir()
	// A fake kubectl that records its argv and exits with a scripted code.
	logPath := filepath.Join(dir, "argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + logPath + "\nexit ${FAKE_KUBECTL_EXIT:-0}\n"
	if err := os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	node := spec.Deploy{Target: "kubevirt", KubeVirtState: &spec.KubeVirtDeployState{VMName: "kv", Namespace: "vms", KubeContext: "ctx"}}

	// exit 0 → the VM exists → ALIVE.
	t.Setenv("FAKE_KUBECTL_EXIT", "0")
	if !ephemeralUnderlyingResourceAlive(t.Context(), nil, "kv", node) {
		t.Fatal("kubectl present + exit 0 must report alive")
	}
	// exit 1 → the VM is gone → NOT alive (reapable).
	t.Setenv("FAKE_KUBECTL_EXIT", "1")
	if ephemeralUnderlyingResourceAlive(t.Context(), nil, "kv", node) {
		t.Fatal("kubectl present + non-zero exit must report not-alive")
	}

	// The probe invoked kubectl with the persisted namespace + context.
	argv, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("fake kubectl was never invoked: %v", err)
	}
	got := string(argv)
	if !strings.Contains(got, "get vm kv -n vms --no-headers --context ctx") {
		t.Fatalf("kubectl argv = %q, want it to carry vm/namespace/context", got)
	}
}
