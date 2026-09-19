package taskpoet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// twTimeLayout is the format that Taskwarrior uses for timestamps
const twTimeLayout = "20060102T150405Z"

func parseTWTime(s string) (time.Time, error) {
	if t, err := time.Parse(twTimeLayout, s); err == nil {
		return t.UTC(), nil
	}
	// Be kind to hand edited files
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid Taskwarrior time %q, expected like 20060102T150405Z", s)
}

func formatTWTime(t time.Time) string {
	return t.UTC().Format(twTimeLayout)
}

// legacyAnnotationKey matches the 'annotation_1234567890' keys that old
// versions of Taskwarrior exported instead of an annotations array
var legacyAnnotationKey = regexp.MustCompile(`^annotation_(\d+)$`)

// ParseOptions controls how a Taskwarrior export is read
type ParseOptions struct {
	// SkipInvalid drops records that can't be converted, with a warning,
	// instead of failing the whole parse
	SkipInvalid bool
	// Now is used when a record has no timestamps at all
	Now time.Time
}

// ParseResult is the outcome of reading a Taskwarrior export
type ParseResult struct {
	Tasks Tasks
	// Skipped is the number of invalid records dropped (see SkipInvalid)
	Skipped  int
	Warnings []string
}

// ParseTaskWarrior reads the output of 'task export'. It accepts a JSON array,
// one JSON object per line, or several of either back to back. Every key we
// don't recognize is kept as a UDA, so nothing in the export is lost.
func ParseTaskWarrior(r io.Reader, opts ParseOptions) (*ParseResult, error) {
	if opts.Now.IsZero() {
		opts.Now = defaultNow()
	}
	c := &twConverter{now: opts.Now, counts: map[string]int{}}
	res := &ParseResult{}
	seen := map[string]bool{}
	dupes := 0

	handle := func(idx int, raw json.RawMessage) error {
		t, err := c.convert(raw)
		if err != nil {
			err = fmt.Errorf("record %d: %w", idx, err)
			if opts.SkipInvalid {
				res.Skipped++
				res.Warnings = append(res.Warnings, fmt.Sprintf("skipped invalid %v", err))
				return nil
			}
			return err
		}
		if seen[t.UUID] {
			dupes++
			return nil
		}
		seen[t.UUID] = true
		res.Tasks = append(res.Tasks, t)
		return nil
	}

	dec := json.NewDecoder(r)
	idx := 0
	for dec.More() {
		var top json.RawMessage
		if err := dec.Decode(&top); err != nil {
			return nil, fmt.Errorf("reading Taskwarrior export: %w", err)
		}
		switch first := firstByte(top); first {
		case '[':
			var items []json.RawMessage
			if err := json.Unmarshal(top, &items); err != nil {
				return nil, fmt.Errorf("reading Taskwarrior export: %w", err)
			}
			for _, item := range items {
				idx++
				if err := handle(idx, item); err != nil {
					return nil, err
				}
			}
		case '{':
			idx++
			if err := handle(idx, top); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("reading Taskwarrior export: expected a JSON array or object, got %q", first)
		}
	}
	if idx == 0 {
		return nil, errors.New("the Taskwarrior export is empty")
	}
	if dupes > 0 {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%d records appeared more than once in the input, only the first of each was kept", dupes))
	}
	res.Warnings = append(res.Warnings, c.warnings()...)
	return res, nil
}

func firstByte(b []byte) byte {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return 0
	}
	return b[0]
}

// twConverter turns one exported record into a Task, and counts the small
// fixups it had to make so they can be reported once
type twConverter struct {
	now    time.Time
	counts map[string]int
}

const (
	noteNoEntry       = "no entry"
	noteOddPriority   = "odd priority"
	noteBadIMask      = "bad imask"
	noteLegacyAnnot   = "legacy annotations"
	noteDedupeDepends = "duplicate depends"
)

func (c *twConverter) note(kind string) {
	c.counts[kind]++
}

func (c *twConverter) warnings() []string {
	msgs := map[string]string{
		noteNoEntry:       "%d tasks had no 'entry' time, used 'modified' or the import time instead",
		noteOddPriority:   "%d tasks had a priority other than H, M or L, it was kept as the 'priority' UDA",
		noteBadIMask:      "%d tasks had an unreadable 'imask', it was dropped",
		noteLegacyAnnot:   "%d tasks used legacy annotation_<epoch> fields, converted to annotations",
		noteDedupeDepends: "%d tasks listed the same dependency more than once, duplicates removed",
	}
	kinds := make([]string, 0, len(c.counts))
	for k := range c.counts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	out := []string{}
	for _, k := range kinds {
		out = append(out, fmt.Sprintf(msgs[k], c.counts[k]))
	}
	return out
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

// decodeStringList reads either a JSON array of strings (Taskwarrior 2.6+) or
// a comma separated string (older versions)
func decodeStringList(v json.RawMessage) ([]string, error) {
	var l []string
	if err := json.Unmarshal(v, &l); err == nil {
		return l, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return nil, fmt.Errorf("expected a list or a comma separated string, got %s", v)
	}
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

// decodeInt reads an integer that may be written as a number or a string
func decodeInt(v json.RawMessage) (int, error) {
	var n int
	if err := json.Unmarshal(v, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return 0, fmt.Errorf("not an integer: %s", v)
	}
	return strconv.Atoi(strings.TrimSpace(s))
}

func decodeUDA(v json.RawMessage) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(v))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *twConverter) convert(raw json.RawMessage) (*Task, error) { //nolint:gocognit,gocyclo,funlen
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}

	t := &Task{}
	var (
		modified   *time.Time
		hasEntry   bool
		statusSeen bool
		legacy     bool
		priority   string
	)
	// Keys are visited in sorted order so errors and UDAs are deterministic
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		if isNull(v) {
			continue
		}
		var err error
		switch k {
		case "id", "urgency":
			// The working set number and urgency are not ours to keep
		case "uuid":
			err = json.Unmarshal(v, &t.UUID)
			t.UUID = strings.ToLower(strings.TrimSpace(t.UUID))
		case "description":
			err = json.Unmarshal(v, &t.Description)
		case "project":
			err = json.Unmarshal(v, &t.Project)
		case "priority":
			err = json.Unmarshal(v, &priority)
		case "recur":
			err = json.Unmarshal(v, &t.Recur)
		case "rtype":
			err = json.Unmarshal(v, &t.RType)
		case "mask":
			err = json.Unmarshal(v, &t.Mask)
		case "parent":
			err = json.Unmarshal(v, &t.Parent)
			t.Parent = strings.ToLower(t.Parent)
		case "imask":
			n, ierr := decodeInt(v)
			if ierr != nil {
				c.note(noteBadIMask)
			} else {
				t.IMask = &n
			}
		case "status":
			var s string
			if err = json.Unmarshal(v, &s); err != nil {
				break
			}
			statusSeen = true
			switch st := Status(strings.ToLower(s)); st {
			case StatusPending, StatusCompleted, StatusDeleted, StatusRecurring:
				t.Status = st
			case "waiting":
				// Older Taskwarrior has a waiting status. For us that's a
				// pending task that has a wait time.
				t.Status = StatusPending
			default:
				err = fmt.Errorf("unknown status %q", s)
			}
		case "tags":
			t.Tags, err = decodeStringList(v)
		case "depends":
			t.Depends, err = decodeStringList(v)
			for i := range t.Depends {
				t.Depends[i] = strings.ToLower(t.Depends[i])
			}
		case "annotations":
			var as []struct {
				Entry       string `json:"entry"`
				Description string `json:"description"`
			}
			if err = json.Unmarshal(v, &as); err != nil {
				break
			}
			for _, a := range as {
				at, terr := parseTWTime(a.Entry)
				if terr != nil {
					err = fmt.Errorf("annotation: %w", terr)
					break
				}
				t.Annotations = append(t.Annotations, Annotation{Entry: at, Description: a.Description})
			}
		case "entry", "modified", "start", "end", "due", "wait", "until", "scheduled", "reviewed":
			var at time.Time
			if at, err = decodeTWTime(v); err != nil {
				break
			}
			switch k {
			case "entry":
				t.Entry, hasEntry = at, true
			case "modified":
				modified = &at
			case "start":
				t.Start = &at
			case "end":
				t.End = &at
			case "due":
				t.Due = &at
			case "wait":
				t.Wait = &at
			case "until":
				t.Until = &at
			case "scheduled":
				t.Scheduled = &at
			case "reviewed":
				t.Reviewed = &at
			}
		case "effort_impact":
			// Our own extension, so that our export imports again
			var n int
			if err = json.Unmarshal(v, &n); err == nil {
				t.EffortImpact = EffortImpact(n)
			}
		default:
			if lm := legacyAnnotationKey.FindStringSubmatch(k); lm != nil {
				var text string
				if err = json.Unmarshal(v, &text); err != nil {
					break
				}
				epoch, _ := strconv.ParseInt(lm[1], 10, 64)
				t.Annotations = append(t.Annotations, Annotation{Entry: time.Unix(epoch, 0).UTC(), Description: text})
				legacy = true
				break
			}
			var u any
			if u, err = decodeUDA(v); err != nil {
				break
			}
			if t.UDA == nil {
				t.UDA = map[string]any{}
			}
			t.UDA[k] = u
		}
		if err != nil {
			return nil, fmt.Errorf("%q: %w", k, err)
		}
	}

	if legacy {
		c.note(noteLegacyAnnot)
	}
	if !statusSeen {
		return nil, errors.New("missing status")
	}
	if t.UUID == "" {
		return nil, errors.New("missing uuid")
	}

	if !hasEntry {
		c.note(noteNoEntry)
		if modified != nil {
			t.Entry = *modified
		} else {
			t.Entry = c.now
		}
	}
	if modified != nil {
		t.Modified = *modified
	} else {
		t.Modified = t.Entry
	}

	if p, err := ParsePriority(priority); err == nil {
		t.Priority = p
	} else {
		// Someone configured their own priority values, don't lose them
		c.note(noteOddPriority)
		if t.UDA == nil {
			t.UDA = map[string]any{}
		}
		t.UDA["priority"] = priority
	}

	if uniq := filterUniqueStrings(t.Depends); len(uniq) != len(t.Depends) {
		c.note(noteDedupeDepends)
		t.Depends = uniq
	}
	sort.SliceStable(t.Annotations, func(i, j int) bool {
		return t.Annotations[i].Entry.Before(t.Annotations[j].Entry)
	})

	if err := t.Validate(); err != nil {
		return nil, err
	}
	return t, nil
}

func decodeTWTime(v json.RawMessage) (time.Time, error) {
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return time.Time{}, fmt.Errorf("expected a time string, got %s", v)
	}
	return parseTWTime(s)
}

// TaskWarriorRecord returns the task as the object 'task export' would print
func (t Task) TaskWarriorRecord() map[string]any {
	rec := map[string]any{
		"uuid":        t.UUID,
		"status":      string(t.Status),
		"description": t.Description,
		"entry":       formatTWTime(t.Entry),
		"modified":    formatTWTime(t.Modified),
	}
	for k, v := range map[string]string{
		"project":  t.Project,
		"priority": string(t.Priority),
		"recur":    t.Recur,
		"rtype":    t.RType,
		"mask":     t.Mask,
		"parent":   t.Parent,
	} {
		if v != "" {
			rec[k] = v
		}
	}
	for k, v := range map[string]*time.Time{
		"start": t.Start, "end": t.End, "due": t.Due, "wait": t.Wait,
		"until": t.Until, "scheduled": t.Scheduled, "reviewed": t.Reviewed,
	} {
		if v != nil {
			rec[k] = formatTWTime(*v)
		}
	}
	if len(t.Tags) > 0 {
		rec["tags"] = t.Tags
	}
	if len(t.Depends) > 0 {
		rec["depends"] = t.Depends
	}
	if t.IMask != nil {
		rec["imask"] = *t.IMask
	}
	if t.EffortImpact != EffortImpactUnset {
		rec["effort_impact"] = int(t.EffortImpact)
	}
	if len(t.Annotations) > 0 {
		as := make([]map[string]string, len(t.Annotations))
		for i, a := range t.Annotations {
			as[i] = map[string]string{"entry": formatTWTime(a.Entry), "description": a.Description}
		}
		rec["annotations"] = as
	}
	for k, v := range t.UDA {
		if _, taken := rec[k]; !taken {
			rec[k] = v
		}
	}
	return rec
}

// WriteTaskWarrior writes the tasks as a Taskwarrior style JSON array, oldest first
func WriteTaskWarrior(w io.Writer, ts Tasks) error {
	sorted := make(Tasks, len(ts))
	copy(sorted, ts)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].Entry.Equal(sorted[j].Entry) {
			return sorted[i].Entry.Before(sorted[j].Entry)
		}
		return sorted[i].UUID < sorted[j].UUID
	})
	if _, err := io.WriteString(w, "[\n"); err != nil {
		return err
	}
	for i, t := range sorted {
		b, err := json.Marshal(t.TaskWarriorRecord())
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		sep := ",\n"
		if i == len(sorted)-1 {
			sep = "\n"
		}
		if _, err := io.WriteString(w, sep); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "]\n")
	return err
}
