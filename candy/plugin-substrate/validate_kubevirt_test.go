package substratekind

import (
	"encoding/json"
	"testing"
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
	if err != nil && err.Error() != "" {
		if containsStr(err.Error(), "unsupported word") {
			t.Fatalf("kubevirt must be a supported status word, got: %v", err)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
