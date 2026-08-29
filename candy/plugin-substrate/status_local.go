package substratekind

// status_local.go — the LOCAL substrate's OpStatus (P14a: relocated verbatim
// from charly/status_collect_local.go). `target: local` deployments (host
// filesystem applies via ShellExecutor, SSH via SSHExecutor) record themselves
// in the install ledger at ~/.config/opencharly/installed/ — the ledger IS the
// authoritative state, there is no running container to inspect. This collector
// reads the ledger (sdk/kit/install_ledger.go — DefaultLedgerPaths /
// ReadDeployRecord / ReadCandyRecord) and reconstructs one row per deploy-id.
// No deploy-cone (FleetConfig/UnifiedFile), no engine — cleanly movable.

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/spec"
)

// localLedgerPaths is the swappable ledger-paths resolver, defaulting to the
// canonical kit.DefaultLedgerPaths. Tests redirect it at a temp dir — mirrors
// the swappable-var pattern used elsewhere for filesystem boundaries.
var localLedgerPaths = kit.DefaultLedgerPaths

// collectLocalStatus serves the local substrate's OpStatusCollect. It reads the
// install ledger (never the engine) and reconstructs one row per deploy-id.
// Absence of the ledger (no local deploy has ever run on this host) yields zero
// rows — the graceful-degradation contract (no error, no rows).
func collectLocalStatus(_ context.Context, req spec.SubstrateStatusRequest) (spec.SubstrateStatusReply, error) {
	paths, err := localLedgerPaths()
	if err != nil {
		return spec.SubstrateStatusReply{}, nil // a resolver error gates the substrate off, not errors the command
	}
	// Refresh the `system:` section of the per-host charly.yml — the host
	// identity snapshot (hostname, distro, kernel, arch, GPU, virtualization,
	// podman). `charly status` is the natural refresh point: it already reads the
	// per-host config, and the snapshot answers "what is this host?" from the
	// unified local config instead of re-probing on every invocation. Best-effort
	// — a detection failure never fails the status command.
	_ = kit.PopulateSystemInfo()
	// The ledger is the `ledger:` section of the per-host charly.yml. Absence
	// of the ledger (no local deploy has ever run on this host) yields zero
	// rows — the graceful-degradation contract (no error, no rows).
	//
	// A MALFORMED ledger is a different thing and must not be reported as an empty
	// one: printing "nothing deployed" for a charly.yml that could not be parsed is
	// the silent-wrong-answer this collector already produced once, when the
	// relocation left paths.Deploys == "" and the os.Stat guard read ENOENT as
	// "absent". The strict listers draw that distinction -- absent is still nil,
	// unparseable is an error naming the path -- and BOTH passes propagate it, where
	// the pre-cutover code propagated on the candy path and swallowed on the deploy
	// path for no stated reason.
	// Read the FULL ledger maps in ONE read of the per-host charly.yml — the
	// per-record ReadDeployRecord/ReadCandyRecord re-read the 310KB config once

	// per record (measured: 186 reads per status run).

	deploys, candies, err := kit.ReadLedger(paths)
	if err != nil {
		return spec.SubstrateStatusReply{}, fmt.Errorf("local ledger: %w", err)
	}

	if len(deploys) == 0 && len(candies) == 0 {

		return spec.SubstrateStatusReply{}, nil

	}

	// deployAgg accumulates per-deploy-id facts gathered across both ledger passes.
	type deployAgg struct {
		candySet   map[string]bool
		latest     string // RFC3339, newest deployed_at seen
		fromRecord bool   // had an explicit DeployRecord
		target     string // DeployRecord.Target ("" for synthesized)
	}
	aggs := map[string]*deployAgg{}
	get := func(id string) *deployAgg {
		a := aggs[id]
		if a == nil {
			a = &deployAgg{candySet: map[string]bool{}}
			aggs[id] = a
		}
		return a
	}

	// Pass 1: explicit DeployRecords in ledger.deploys (in-memory — read once above).
	for _, rec := range deploys {
		a := get(rec.DeployID)
		a.fromRecord = true
		a.target = rec.Target
		for _, ln := range rec.Candy {
			a.candySet[ln] = true
		}
		for _, ln := range rec.AddCandy {
			a.candySet[ln] = true
		}
		a.latest = newerTimestamp(a.latest, rec.DeployedAt)
	}

	// Pass 2: CandyRecords in ledger.candies — attribute each candy to every
	// deploy-id in its deployed_by set (in-memory — read once above).
	for _, rec := range candies {
		for _, id := range rec.DeployedBy {
			a := get(id)
			a.candySet[rec.Candy] = true
			a.latest = newerTimestamp(a.latest, rec.DeployedAt)
		}
	}

	rows := make([]spec.DeploymentStatus, 0, len(aggs))
	for id, a := range aggs {
		rows = append(rows, spec.DeploymentStatus{
			Kind:      spec.SubstrateLocal,
			Source:    "ledger",
			Image:     localDeployLabel(len(a.candySet)),
			Status:    "applied",
			Uptime:    formatLedgerTimestamp(a.latest),
			Container: id,
			RunMode:   req.RunMode,
		})
	}

	// Deterministic ordering by deploy-id; the host re-sorts the merged set
	// across substrates, but a stable order here keeps single-substrate output
	// predictable.
	sort.Slice(rows, func(i, j int) bool { return rows[i].Container < rows[j].Container })
	return spec.SubstrateStatusReply{Rows: rows}, nil
}

// localDeployLabel renders the IMAGE-cell text for a local deploy, which has no
// container image. It reports the applied-candy count instead.
func localDeployLabel(n int) string {
	if n == 1 {
		return "local (1 candy)"
	}
	return fmt.Sprintf("local (%d candies)", n)
}

// newerTimestamp returns whichever of two RFC3339 timestamps is later. A
// non-empty value beats empty; an unparseable value is treated as older than a
// parseable one so a malformed record never masks a good deployed_at.
func newerTimestamp(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	ta, errA := time.Parse(time.RFC3339, a)
	tb, errB := time.Parse(time.RFC3339, b)
	switch {
	case errA != nil && errB != nil:
		return a
	case errA != nil:
		return b
	case errB != nil:
		return a
	}
	if tb.After(ta) {
		return b
	}
	return a
}

// formatLedgerTimestamp renders a ledger deployed_at (RFC3339) for the Uptime
// cell. Ledger deploys have no "uptime" — the filesystem state persists — so we
// surface the apply time as an absolute UTC instant. An empty or unparseable
// value yields "".
func formatLedgerTimestamp(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return "deployed " + t.UTC().Format("2006-01-02 15:04 MST")
}
