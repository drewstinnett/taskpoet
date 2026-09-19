package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// Interval is how long to leave between asking GitHub, and between
	// telling the user about the same release
	Interval = 6 * time.Hour

	// fetchTimeout is how long a lazy check may take altogether
	fetchTimeout = 3 * time.Second
)

// state is what is remembered between runs, in one small file
type state struct {
	// CheckedAt is when we last asked, whether or not it worked, so a broken
	// network costs one attempt per Interval and not one per command
	CheckedAt  time.Time `json:"checked_at"`
	Latest     string    `json:"latest,omitempty"`
	NotifiedAt time.Time `json:"notified_at"`
}

// Checker keeps track of the newest release
type Checker struct {
	Source    *Source
	Current   Version
	StatePath string
	Interval  time.Duration
	// Now is the time, and only different in tests
	Now func() time.Time
}

// NewChecker is a Checker with the usual interval. The state lives in the
// user's cache directory, it is fine if it goes missing.
func NewChecker(src *Source, current Version) (*Checker, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return &Checker{
		Source:    src,
		Current:   current,
		StatePath: filepath.Join(dir, "taskpoet", "update-check.json"),
		Interval:  Interval,
		Now:       time.Now,
	}, nil
}

// load never fails: a missing or damaged file is the same as never having checked
func (c *Checker) load() state {
	var st state
	if b, err := os.ReadFile(c.StatePath); err == nil {
		if json.Unmarshal(b, &st) != nil {
			return state{}
		}
	}
	return st
}

// save is best effort, the worst that comes of a failure is a repeated check.
// The file is replaced in one rename, so a reader never sees half of it.
func (c *Checker) save(st state) {
	b, err := json.Marshal(st)
	if err != nil {
		return
	}
	dir := filepath.Dir(c.StatePath)
	if os.MkdirAll(dir, 0o750) != nil {
		return
	}
	f, err := os.CreateTemp(dir, ".update-check-*")
	if err != nil {
		return
	}
	_, werr := f.Write(b)
	if cerr := f.Close(); werr != nil || cerr != nil || os.Rename(f.Name(), c.StatePath) != nil {
		_ = os.Remove(f.Name())
	}
}

func (c *Checker) due(st state) bool {
	return c.Now().Sub(st.CheckedAt) >= c.Interval
}

// newer is the release we know of if it is later than the running one
func (c *Checker) newer(st state) (Version, bool) {
	v, err := ParseVersion(st.Latest)
	if err != nil || !v.After(c.Current) {
		return Version{}, false
	}
	return v, true
}

// Refresh asks GitHub for the newest release right now, and remembers the answer
func (c *Checker) Refresh(ctx context.Context) (Version, error) {
	st := c.load()
	st.CheckedAt = c.Now()
	c.save(st) // before asking: if we're cut off, this still counts as a try
	v, err := c.Source.Latest(ctx)
	if err != nil {
		return Version{}, err
	}
	st = c.load()
	st.Latest = v.Tag()
	c.save(st)
	return v, nil
}

// Lazy is a check that happens in the background
type Lazy struct {
	c      *Checker
	done   chan struct{} // closed when the refresh has finished, nil if there is none
	cancel context.CancelFunc
}

// Start begins a check if it is time for one. It does not wait for it.
func (c *Checker) Start() *Lazy {
	l := &Lazy{c: c}
	if !c.due(c.load()) {
		return l
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	l.cancel = cancel
	l.done = make(chan struct{})
	go func() {
		defer close(l.done)
		_, _ = c.Refresh(ctx)
	}()
	return l
}

// Wait gives a check that is under way up to budget to finish, then says
// whether there is a newer release the user has not heard about lately. Being
// told counts: it won't say yes again for another Interval. Safe on nil.
func (l *Lazy) Wait(budget time.Duration) (Version, bool) {
	if l == nil {
		return Version{}, false
	}
	if l.done != nil {
		select {
		case <-l.done:
		case <-time.After(budget):
		}
		l.cancel()
	}
	st := l.c.load()
	v, ok := l.c.newer(st)
	if !ok || l.c.Now().Sub(st.NotifiedAt) < l.c.Interval {
		return Version{}, false
	}
	st.NotifiedAt = l.c.Now()
	l.c.save(st)
	return v, true
}
