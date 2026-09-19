package taskpoet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
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
		// The binary is whatever the user pointed --task-bin at, on purpose
		cmd := exec.CommandContext(ctx, bin, //nolint:gosec
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
			return nil, exportError(bin, filter, err, stderr.String())
		}
		out.Write(b)
		out.WriteByte('\n')
	}
	return out.Bytes(), nil
}

// exportError explains a failed 'task export'. When there is no Taskwarrior to
// run, it points at importing an export file instead.
func exportError(bin, filter string, err error, stderr string) error {
	// Not on the PATH, or an explicit path that isn't there
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("can't find Taskwarrior (%q). Install it, point --task-bin at it, or import an export made elsewhere with 'task export > tw.json' using 'taskpoet import tw.json'", bin)
	}
	if msg := strings.TrimSpace(stderr); msg != "" {
		return fmt.Errorf("running '%v %v export': %w (%v)", bin, filter, err, msg)
	}
	return fmt.Errorf("running '%v %v export': %w", bin, filter, err)
}
