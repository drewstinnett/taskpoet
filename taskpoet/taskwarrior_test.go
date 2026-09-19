package taskpoet

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	uuidReport   = "11111111-1111-4111-8111-111111111111"
	uuidSend     = "22222222-2222-4222-8222-222222222222"
	uuidMilk     = "33333333-3333-4333-8333-333333333333"
	uuidDeleted  = "44444444-4444-4444-8444-444444444444"
	uuidTplWeek  = "55555555-5555-4555-8555-555555555555"
	uuidInstDone = "66666666-6666-4666-8666-666666666666"
	uuidInstPend = "77777777-7777-4777-8777-777777777777"
	uuidTplChain = "88888888-8888-4888-8888-888888888888"
	uuidInstChn  = "99999999-9999-4999-8999-999999999999"
	uuidUnicode  = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	uuidEffort   = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func twTime(t *testing.T, s string) time.Time {
	t.Helper()
	got, err := parseTWTime(s)
	require.NoError(t, err)
	return got
}

func mustParseFixture(t *testing.T, name string, opts ParseOptions) *ParseResult {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "tw", name))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	res, err := ParseTaskWarrior(f, opts)
	require.NoError(t, err)
	return res
}

func byUUID(ts Tasks) map[string]*Task {
	m := map[string]*Task{}
	for _, t := range ts {
		m[t.UUID] = t
	}
	return m
}

func TestParseTWTime(t *testing.T) {
	got, err := parseTWTime("20240102T090000Z")
	require.NoError(t, err)
	require.Equal(t, time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC), got)
	require.Equal(t, "20240102T090000Z", formatTWTime(got))

	got, err = parseTWTime("2024-01-02T09:00:00+02:00")
	require.NoError(t, err)
	require.Equal(t, time.Date(2024, 1, 2, 7, 0, 0, 0, time.UTC), got)

	for _, bad := range []string{"", "yesterday", "2024-01-02", "20240102"} {
		_, err := parseTWTime(bad)
		require.Error(t, err, bad)
	}
}

func TestParseCanonical(t *testing.T) {
	res := mustParseFixture(t, "canonical.json", ParseOptions{})
	require.Len(t, res.Tasks, 11)
	require.Empty(t, res.Warnings)
	require.Zero(t, res.Skipped)
	all := byUUID(res.Tasks)

	t.Run("rich pending task", func(t *testing.T) {
		got := all[uuidReport]
		require.Equal(t, StatusPending, got.Status)
		require.Equal(t, "Write the quarterly report", got.Description)
		require.Equal(t, "work.reports", got.Project)
		require.Equal(t, PriorityHigh, got.Priority)
		require.Equal(t, []string{"writing", "next"}, got.Tags)
		require.Equal(t, twTime(t, "20240102T090000Z"), got.Entry)
		require.Equal(t, twTime(t, "20240110T101500Z"), got.Modified)
		require.Equal(t, twTime(t, "20240131T170000Z"), *got.Due)
		require.Equal(t, twTime(t, "20240109T080000Z"), *got.Start)
		require.Equal(t, []Annotation{
			{Entry: twTime(t, "20240103T120000Z"), Description: "Ask finance for the numbers"},
			{Entry: twTime(t, "20240108T090000Z"), Description: `Numbers arrived, "final" version pending`},
		}, got.Annotations)
		// id and urgency are not kept, everything unknown is
		require.Equal(t, map[string]any{
			"estimate":    "2h",
			"storypoints": json.Number("5"),
			"signoff":     "20240201T000000Z",
		}, got.UDA)
	})

	t.Run("waiting and scheduled", func(t *testing.T) {
		got := all[uuidSend]
		require.Equal(t, StatusPending, got.Status)
		require.Equal(t, []string{uuidReport}, got.Depends)
		require.Equal(t, twTime(t, "20300101T000000Z"), *got.Wait)
		require.Equal(t, twTime(t, "20300102T090000Z"), *got.Scheduled)
		require.True(t, got.IsWaiting(time.Now()))
	})

	t.Run("completed and deleted", func(t *testing.T) {
		got := all[uuidMilk]
		require.Equal(t, StatusCompleted, got.Status)
		require.Equal(t, twTime(t, "20240104T180000Z"), *got.End)
		got = all[uuidDeleted]
		require.Equal(t, StatusDeleted, got.Status)
		require.Equal(t, twTime(t, "20240105T090000Z"), *got.End)
	})

	t.Run("periodic template and instances", func(t *testing.T) {
		tpl := all[uuidTplWeek]
		require.Equal(t, StatusRecurring, tpl.Status)
		require.Equal(t, "weekly", tpl.Recur)
		require.Equal(t, "periodic", tpl.RType)
		require.Equal(t, "+-", tpl.Mask)
		require.Equal(t, twTime(t, "20241231T000000Z"), *tpl.Until)
		require.Equal(t, twTime(t, "20240108T090000Z"), *tpl.Due)
		require.Equal(t, PriorityLow, tpl.Priority)

		done := all[uuidInstDone]
		require.Equal(t, StatusCompleted, done.Status)
		require.Equal(t, uuidTplWeek, done.Parent)
		require.Equal(t, 0, *done.IMask)
		pend := all[uuidInstPend]
		require.Equal(t, uuidTplWeek, pend.Parent)
		require.Equal(t, 1, *pend.IMask)
	})

	t.Run("chained template without a mask", func(t *testing.T) {
		tpl := all[uuidTplChain]
		require.Equal(t, StatusRecurring, tpl.Status)
		require.Equal(t, "chained", tpl.RType)
		require.Equal(t, "3months", tpl.Recur)
		require.Empty(t, tpl.Mask)
		inst := all[uuidInstChn]
		require.Equal(t, uuidTplChain, inst.Parent)
		require.Equal(t, 0, *inst.IMask)
	})

	t.Run("unicode and reviewed", func(t *testing.T) {
		got := all[uuidUnicode]
		require.Equal(t, `Café ☕ – résumé "quotes" and 日本語`, got.Description)
		require.Equal(t, twTime(t, "20240105T100000Z"), *got.Reviewed)
	})

	t.Run("our own extension", func(t *testing.T) {
		got := all[uuidEffort]
		require.Equal(t, EffortImpactMedium, got.EffortImpact)
		require.Empty(t, got.UDA)
		require.Equal(t, []string{uuidReport, uuidSend}, got.Depends)
	})
}

func TestParseLegacyShapes(t *testing.T) {
	now := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
	res := mustParseFixture(t, "legacy.json", ParseOptions{Now: now})
	require.Len(t, res.Tasks, 11)
	all := byUUID(res.Tasks)
	id := func(n string) string { return "aaaaaaaa-0000-4000-8000-0000000000" + n }

	t.Run("waiting status and comma separated lists", func(t *testing.T) {
		got := all[id("01")]
		require.Equal(t, StatusPending, got.Status)
		require.Equal(t, twTime(t, "20990101T000000Z"), *got.Wait)
		require.Equal(t, []string{"home", "errand", "urgent"}, got.Tags)
		require.Equal(t, []string{id("02")}, got.Depends, "duplicate dependency removed")
	})

	t.Run("annotation_ keys become annotations, oldest first", func(t *testing.T) {
		got := all[id("02")]
		require.Equal(t, []Annotation{
			{Entry: time.Unix(1577836800, 0).UTC(), Description: "older note"},
			{Entry: time.Unix(1577923200, 0).UTC(), Description: "first note"},
		}, got.Annotations)
		require.Empty(t, got.Project, "null is the same as missing")
		require.Empty(t, got.UDA, "annotation_ keys are not UDAs")
	})

	t.Run("missing entry falls back to modified", func(t *testing.T) {
		got := all[id("03")]
		want := twTime(t, "20210505T050505Z")
		require.Equal(t, want, got.Entry)
		require.Equal(t, want, got.Modified)
	})

	t.Run("custom priority is kept as a UDA", func(t *testing.T) {
		got := all[id("04")]
		require.Equal(t, PriorityNone, got.Priority)
		require.Equal(t, map[string]any{"priority": "urgent"}, got.UDA)
	})

	t.Run("string imask and old style mask", func(t *testing.T) {
		require.Equal(t, "++-", all[id("05")].Mask)
		require.Equal(t, 2, *all[id("06")].IMask)
		require.Equal(t, id("05"), all[id("06")].Parent)
	})

	joined := strings.Join(res.Warnings, "\n")
	for _, want := range []string{
		"1 tasks had no 'entry'",
		"1 tasks had a priority other than H, M or L",
		"1 tasks used legacy annotation_<epoch>",
		"1 tasks listed the same dependency more than once",
	} {
		require.Contains(t, joined, want)
	}
}

func TestParseEntryFallsBackToNow(t *testing.T) {
	now := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
	res, err := ParseTaskWarrior(strings.NewReader(`[{"description":"x","status":"pending","uuid":"u1"}]`), ParseOptions{Now: now})
	require.NoError(t, err)
	require.Equal(t, now, res.Tasks[0].Entry)
	require.Equal(t, now, res.Tasks[0].Modified)
}

func TestParseNDJSON(t *testing.T) {
	res := mustParseFixture(t, "ndjson.jsonl", ParseOptions{})
	require.Len(t, res.Tasks, 3)
	require.Equal(t, StatusPending, res.Tasks[0].Status)
	require.Equal(t, StatusCompleted, res.Tasks[1].Status)
	require.Equal(t, StatusDeleted, res.Tasks[2].Status)
}

func TestParseConcatenatedArrays(t *testing.T) {
	// This is what running 'task export' once per status produces
	in := `[{"description":"a","status":"pending","uuid":"u1","entry":"20240101T000000Z"}]
[{"description":"b","status":"completed","uuid":"u2","entry":"20240101T000000Z"}]
[]
{"description":"c","status":"deleted","uuid":"u3","entry":"20240101T000000Z"}`
	res, err := ParseTaskWarrior(strings.NewReader(in), ParseOptions{})
	require.NoError(t, err)
	require.Len(t, res.Tasks, 3)
}

func TestParseDuplicateUUID(t *testing.T) {
	in := `[{"description":"first","status":"pending","uuid":"u1","entry":"20240101T000000Z"},
	{"description":"second","status":"pending","uuid":"u1","entry":"20240101T000000Z"}]`
	res, err := ParseTaskWarrior(strings.NewReader(in), ParseOptions{})
	require.NoError(t, err)
	require.Len(t, res.Tasks, 1)
	require.Equal(t, "first", res.Tasks[0].Description)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "1 records appeared more than once")
}

func TestParseInvalid(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"unknown status":      {`[{"description":"x","status":"bogus","uuid":"u1"}]`, `unknown status "bogus"`},
		"missing status":      {`[{"description":"x","uuid":"u1"}]`, "missing status"},
		"missing uuid":        {`[{"description":"x","status":"pending"}]`, "missing uuid"},
		"missing description": {`[{"status":"pending","uuid":"u1"}]`, "missing description"},
		"bad time":            {`[{"description":"x","status":"pending","uuid":"u1","due":"tomorrow"}]`, `"due"`},
		"time not a string":   {`[{"description":"x","status":"pending","uuid":"u1","due":5}]`, `"due"`},
		"bad annotation":      {`[{"description":"x","status":"pending","uuid":"u1","annotations":[{"entry":"nope","description":"n"}]}]`, "annotation"},
		"depends wrong type":  {`[{"description":"x","status":"pending","uuid":"u1","depends":5}]`, `"depends"`},
		"self dependency":     {`[{"description":"x","status":"pending","uuid":"u1","depends":["u1"]}]`, "depend on itself"},
		"not an object":       {`["hello"]`, "not a JSON object"},
		"top level scalar":    {`"hello"`, "expected a JSON array or object"},
		"broken json":         {`[{"description":`, "reading Taskwarrior export"},
		"empty":               {``, "empty"},
		"empty array":         {`[]`, "empty"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseTaskWarrior(strings.NewReader(tt.in), ParseOptions{})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseSkipInvalid(t *testing.T) {
	in := `[
	{"description":"good","status":"pending","uuid":"u1","entry":"20240101T000000Z"},
	{"description":"bad","status":"bogus","uuid":"u2"},
	{"description":"also good","status":"pending","uuid":"u3","entry":"20240101T000000Z"}]`
	res, err := ParseTaskWarrior(strings.NewReader(in), ParseOptions{SkipInvalid: true})
	require.NoError(t, err)
	require.Len(t, res.Tasks, 2)
	require.Equal(t, 1, res.Skipped)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "skipped invalid record 2")
}

// normalize decodes a JSON array of tasks into a map by uuid, without the
// fields we never keep
func normalize(t *testing.T, b []byte) map[string]map[string]any {
	t.Helper()
	var recs []map[string]any
	require.NoError(t, json.Unmarshal(b, &recs))
	out := map[string]map[string]any{}
	for _, r := range recs {
		delete(r, "id")
		delete(r, "urgency")
		out[r["uuid"].(string)] = r
	}
	return out
}

func TestExportRoundTrip(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tw", "canonical.json"))
	require.NoError(t, err)
	res, err := ParseTaskWarrior(bytes.NewReader(raw), ParseOptions{})
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, WriteTaskWarrior(&out, res.Tasks))
	require.Equal(t, normalize(t, raw), normalize(t, out.Bytes()))

	// The output is a well formed export in its own right
	again, err := ParseTaskWarrior(bytes.NewReader(out.Bytes()), ParseOptions{})
	require.NoError(t, err)
	require.Equal(t, byUUID(res.Tasks), byUUID(again.Tasks))
}

func TestExportOrder(t *testing.T) {
	a := MustNewTask("newer", WithUUID("b"), WithEntry(testEpoch.Add(time.Hour)))
	b := MustNewTask("older", WithUUID("a"), WithEntry(testEpoch))
	var out bytes.Buffer
	require.NoError(t, WriteTaskWarrior(&out, Tasks{a, b}))
	var recs []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &recs))
	require.Equal(t, "older", recs[0]["description"])
	require.Equal(t, "newer", recs[1]["description"])

	out.Reset()
	require.NoError(t, WriteTaskWarrior(&out, nil))
	require.JSONEq(t, `[]`, out.String())
}

// fakeTask writes a stand-in for the 'task' binary that answers 'export' with
// canned output per status filter
func fakeTask(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs a shell")
	}
	bin := filepath.Join(t.TempDir(), "task")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o700)) // nolint:gosec
	return bin
}

func TestRunTaskWarriorExportFailure(t *testing.T) {
	bin := fakeTask(t, `
for a in "$@"; do
  case "$a" in
    status:completed) echo 'Something is badly wrong' >&2; exit 2;;
  esac
done
echo '[]'
`)
	// A required status failing is a real failure
	_, err := RunTaskWarriorExport(context.Background(), bin)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status:completed")
	require.Contains(t, err.Error(), "Something is badly wrong")
}

func TestRunTaskWarriorExportWithoutWaitingStatus(t *testing.T) {
	bin := fakeTask(t, `
for a in "$@"; do
  case "$a" in
    status:waiting) echo 'The filter is not understood' >&2; exit 2;;
    status:pending) echo '[{"description":"p","status":"pending","uuid":"u1","entry":"20240101T000000Z"}]'; exit 0;;
  esac
done
exit 1
`)
	// Newer versions may not know status:waiting, and that isn't fatal
	raw, err := RunTaskWarriorExport(context.Background(), bin)
	require.NoError(t, err)
	res, err := ParseTaskWarrior(bytes.NewReader(raw), ParseOptions{})
	require.NoError(t, err)
	require.Len(t, res.Tasks, 1)
}

func TestRunTaskWarriorExportAllStatuses(t *testing.T) {
	bin := fakeTask(t, `
for a in "$@"; do
  case "$a" in
    status:pending)   echo '[{"description":"p","status":"pending","uuid":"u1","entry":"20240101T000000Z"}]'; exit 0;;
    status:waiting)   echo '[{"description":"w","status":"waiting","uuid":"u4","entry":"20240101T000000Z","wait":"20990101T000000Z"}]'; exit 0;;
    status:completed) echo '[{"description":"c","status":"completed","uuid":"u2","entry":"20240101T000000Z"}]'; exit 0;;
    status:deleted)   exit 1;;
    status:recurring) echo '[{"description":"r","status":"recurring","uuid":"u3","entry":"20240101T000000Z","recur":"daily"},{"description":"p","status":"pending","uuid":"u1","entry":"20240101T000000Z"}]'; exit 0;;
  esac
done
exit 9
`)
	raw, err := RunTaskWarriorExport(context.Background(), bin)
	require.NoError(t, err)
	res, err := ParseTaskWarrior(bytes.NewReader(raw), ParseOptions{})
	require.NoError(t, err)
	// Nothing deleted (exit 1 is 'no matches'), and u1 turned up twice
	require.Len(t, res.Tasks, 4)
	require.Contains(t, strings.Join(res.Warnings, "\n"), "1 records appeared more than once")
	all := byUUID(res.Tasks)
	require.Equal(t, StatusRecurring, all["u3"].Status)
	require.NotNil(t, all["u4"].Wait)
}

func TestRunTaskWarriorExportMissingBinary(t *testing.T) {
	_, err := RunTaskWarriorExport(context.Background(), filepath.Join(t.TempDir(), "no-such-task"))
	require.Error(t, err)
}
