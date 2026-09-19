package taskpoet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// taskWarriorExportFilters are the exports we ask Taskwarrior for. A plain
// 'task export' has not always meant everything (versions differ on whether
// completed, deleted, recurring or waiting tasks are included), so each status
// is asked for by name. Overlap between them is fine, the importer keeps the
// first copy of any task it sees twice.
var taskWarriorExportFilters = []struct {
	filter string
	// optional filters may fail without failing the import
	optional bool
}{
	{filter: "status:pending"},
	// Not a real status in newer versions, they may not understand it
	{filter: "status:waiting", optional: true},
	{filter: "status:completed"},
	{filter: "status:deleted"},
	{filter: "status:recurring"},
}

// RunTaskWarriorExport runs 'task export' for every status using the given
// Taskwarrior binary, and returns the combined output for ParseTaskWarrior.
// Taskwarrior's own environment (TASKRC, TASKDATA) is honored.
func RunTaskWarriorExport(ctx context.Context, bin string) ([]byte, error) {
	var out bytes.Buffer
	for _, f := range taskWarriorExportFilters {
		filter := f.filter
		cmd := exec.CommandContext(ctx, bin,
			"rc.json.array=on",
			"rc.verbose=nothing",
			"rc.confirmation=off",
			"rc.hooks=off",
			filter,
			"export",
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		b, err := cmd.Output()
		if err != nil && f.optional && ctx.Err() == nil {
			var execErr *exec.Error
			if !errors.As(err, &execErr) {
				continue
			}
		}
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && len(bytes.TrimSpace(b)) == 0 {
				// Taskwarrior exits 1 when nothing matches
				continue
			}
			return nil, fmt.Errorf("running '%v %v export': %w (%v)", bin, filter, err, strings.TrimSpace(stderr.String()))
		}
		out.Write(b)
		out.WriteByte('\n')
	}
	return out.Bytes(), nil
}
