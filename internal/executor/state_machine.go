package executor

import (
	"time"

	"aios/internal/dsl"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func upsertNodeRun(rec *dsl.RunRecord, nr dsl.NodeRun) {
	for i := range rec.NodeRuns {
		if rec.NodeRuns[i].NodeID == nr.NodeID {
			rec.NodeRuns[i] = nr
			return
		}
	}
	rec.NodeRuns = append(rec.NodeRuns, nr)
}

