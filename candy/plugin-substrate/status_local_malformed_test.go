package substratekind

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/spec"
)

// A malformed per-host charly.yml must surface as an ERROR, never as "nothing
// deployed". This collector already shipped that silent-wrong-answer once: the
// ledger relocation left LedgerPaths.Deploys == "", os.Stat("") returned ENOENT,
// and the availability guard read a relocated ledger as an absent one. These tests
// pin the distinction so the next refactor cannot quietly reintroduce it.

func redirectMalformedLedger(t *testing.T, body string) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "charly.yml")
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	prev := localLedgerPaths
	localLedgerPaths = func() (*kit.LedgerPaths, error) {
		return &kit.LedgerPaths{ConfigFile: cfg, LockFile: cfg + ".lock"}, nil
	}
	t.Cleanup(func() { localLedgerPaths = prev })
}

func TestCollectLocalStatus_MalformedLedgerErrors(t *testing.T) {
	redirectMalformedLedger(t, "ledger: [this is not a mapping\n")

	reply, err := collectLocalStatus(context.Background(), spec.SubstrateStatusRequest{RunMode: "quadlet"})
	if err == nil {
		t.Fatalf("malformed ledger reported as success with %d rows — the silent wrong answer", len(reply.Rows))
	}
	if !strings.Contains(err.Error(), "local ledger") {
		t.Errorf("error does not name the ledger: %v", err)
	}
}

// The graceful-degradation contract still holds for a genuinely ABSENT ledger:
// no local deploy has ever run here, so zero rows and no error.
func TestCollectLocalStatus_AbsentLedgerStaysQuiet(t *testing.T) {
	prev := localLedgerPaths
	cfg := filepath.Join(t.TempDir(), "absent.yml")
	localLedgerPaths = func() (*kit.LedgerPaths, error) {
		return &kit.LedgerPaths{ConfigFile: cfg, LockFile: cfg + ".lock"}, nil
	}
	t.Cleanup(func() { localLedgerPaths = prev })

	reply, err := collectLocalStatus(context.Background(), spec.SubstrateStatusRequest{RunMode: "quadlet"})
	if err != nil {
		t.Fatalf("absent ledger must not error: %v", err)
	}
	if len(reply.Rows) != 0 {
		t.Errorf("rows = %d, want 0", len(reply.Rows))
	}
}
