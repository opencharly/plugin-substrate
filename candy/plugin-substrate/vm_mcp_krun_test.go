package substratekind

// vm_mcp_krun_test.go — two more behaviour tests of the vm body, moved here with the body
// (opencharly/spec#304 unit 8): the mcp_provide gate (three cases, including the url-required
// rejection that proves the field binds #CandyMCPProvide rather than an open passthrough) and the
// krun defaults materialization (the required-with-default fields carry the measured values in
// the VALUE, not in a consumer-side literal).
//
// They used to compile spec's shipped schema; the def lives in THIS plugin's CUE module now, so
// the harness composes what the host's gate composes and asserts against that value.

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
)

func TestVm_MCPProvideSchemaParse(t *testing.T) {
	schema := composedBodySchema(t)
	def := schema.LookupPath(cue.ParsePath("#Vm"))
	if err := def.Err(); err != nil {
		t.Fatalf("lookup #Vm: %v", err)
	}

	for _, tc := range []struct {
		name   string
		value  string
		reject bool
	}{
		{
			name:  "kind:vm charly.yml with mcp_provide parses",
			value: `{source: {kind: "cloud_image", url: "https://example.com/os.qcow2"}, mcp_provide: [{name: "charly-mcp", url: "http://127.0.0.1:18765/mcp"}, {name: "charly-sse", url: "http://127.0.0.1:18766/sse", transport: "sse"}]}`,
		},
		{
			name:  "mcp_provide without transport defaults to http",
			value: `{source: {kind: "cloud_image", url: "https://example.com/os.qcow2"}, mcp_provide: [{name: "charly-mcp", url: "http://127.0.0.1:18765/mcp"}]}`,
		},
		{
			// #CandyMCPProvide requires url — proves the field binds the candy
			// shape rather than an open passthrough.
			name:   "mcp_provide entry missing url rejected",
			value:  `{source: {kind: "cloud_image", url: "https://example.com/os.qcow2"}, mcp_provide: [{name: "charly-mcp"}]}`,
			reject: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			val := schema.Context().CompileString(tc.value)
			if err := val.Err(); err != nil {
				t.Fatalf("compile value: %v", err)
			}
			got := def.Unify(val).Validate(cue.Concrete(true), cue.Final())
			if tc.reject && got == nil {
				t.Errorf("value ACCEPTED, want rejection: %s", tc.value)
			}
			if !tc.reject && got != nil {
				t.Errorf("value rejected: %s — %v", tc.value, got)
			}
			if tc.reject && got != nil && !strings.Contains(got.Error(), "url") {
				t.Logf("note: rejection message does not name the url field: %v", got)
			}
		})
	}
}

func TestVmKrunDefaultsMaterialize(t *testing.T) {
	def, ctx := composedVm(t)
	const body = `{backend: "krun", source: {kind: "container_disk", image: "registry.example/disk@sha256:0000000000000000000000000000000000000000000000000000000000000000"}, krun: {}}`
	v := def.Unify(ctx.CompileString(body))
	if err := v.Validate(cue.Concrete(true)); err != nil {
		t.Fatalf("the krun defaults do not materialize under a concrete validation: %v", err)
	}
	// The fields are required-with-default, so an empty krun block carries the
	// measured values in the value itself — not a consumer-side literal.
	for _, tc := range []struct {
		path string
		want int64
	}{
		{"krun.gpu_flags", 962},
		{"krun.ram_headroom_mib", 256},
	} {
		got, err := v.LookupPath(cue.ParsePath(tc.path)).Int64()
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if got != tc.want {
			t.Errorf("%s = %d, want %d", tc.path, got, tc.want)
		}
	}
	for _, tc := range []struct {
		path, want string
	}{
		{"krun.network", "tsi"},
		{"krun.kernel", "stock"},
		{"krun.gpu", "off"},
	} {
		got, err := v.LookupPath(cue.ParsePath(tc.path)).String()
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if got != tc.want {
			t.Errorf("%s = %q, want %q", tc.path, got, tc.want)
		}
	}
	if got, err := v.LookupPath(cue.ParsePath("krun.nested_virt")).Bool(); err != nil || got {
		t.Errorf("krun.nested_virt = %v, %v; want false", got, err)
	}
}
