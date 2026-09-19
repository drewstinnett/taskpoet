package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	good := map[string]Version{
		"v2.1.0":   {2, 1, 0},
		"2.1.0":    {2, 1, 0},
		"v0.10.12": {0, 10, 12},
	}
	for in, want := range good {
		got, err := ParseVersion(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}
	// 'git describe' after a tag, a dirty tree, a pre-release and plain junk
	for _, in := range []string{"", "dev", "v2.1.0-3-gabc123", "v2.1.0-dirty", "v2.1.0-rc.1", "v2.1", "v01.2.3", "v2.1.0.1", "vv2.1.0"} {
		_, err := ParseVersion(in)
		require.Error(t, err, in)
	}
}

func TestVersionAfter(t *testing.T) {
	v := func(s string) Version {
		got, err := ParseVersion(s)
		require.NoError(t, err)
		return got
	}
	require.True(t, v("2.0.1").After(v("2.0.0")))
	require.True(t, v("2.1.0").After(v("2.0.9")))
	require.True(t, v("3.0.0").After(v("2.9.9")))
	require.True(t, v("2.10.0").After(v("2.9.0")), "numbers, not text")
	require.False(t, v("2.0.0").After(v("2.0.0")))
	require.False(t, v("1.9.9").After(v("2.0.0")))
	require.Equal(t, "v2.1.0", v("2.1.0").Tag())
	require.Equal(t, "2.1.0", v("v2.1.0").String())
}

// fakeGitHub serves one release. latest is what the API says, and hits counts
// the calls to the API.
type fakeGitHub struct {
	*httptest.Server
	hits atomic.Int32
}

func newFakeGitHub(t *testing.T, latest string, files map[string][]byte) *fakeGitHub {
	t.Helper()
	f := &fakeGitHub{}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		f.hits.Add(1)
		fmt.Fprintf(w, `{"tag_name": %q, "name": "whatever"}`, latest)
	})
	mux.HandleFunc("/o/r/releases/download/", func(w http.ResponseWriter, r *http.Request) {
		b, ok := files[filepath.Base(r.URL.Path)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(b)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeGitHub) source() *Source {
	return &Source{API: f.URL, Download: f.URL, Repo: "o/r", Client: f.Client(), UserAgent: "test"}
}

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

// newChecker is a checker for a taskpoet that is v2.0.0
func newChecker(t *testing.T, src *Source, clk *clock) *Checker {
	t.Helper()
	return &Checker{
		Source:    src,
		Current:   Version{2, 0, 0},
		StatePath: filepath.Join(t.TempDir(), "sub", "update-check.json"),
		Interval:  6 * time.Hour,
		Now:       clk.Now,
	}
}

func TestLatest(t *testing.T) {
	gh := newFakeGitHub(t, "v2.3.4", nil)
	v, err := gh.source().Latest(context.Background())
	require.NoError(t, err)
	require.Equal(t, Version{2, 3, 4}, v)

	// a tag that isn't a plain version can't be trusted into a download URL
	bad := newFakeGitHub(t, "v2.3.4/../../x", nil)
	_, err = bad.source().Latest(context.Background())
	require.Error(t, err)
}

func TestLazyChecksAtMostOncePerInterval(t *testing.T) {
	gh := newFakeGitHub(t, "v2.1.0", nil)
	clk := &clock{now: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)}
	c := newChecker(t, gh.source(), clk)

	// first run: nothing known yet, so it asks, and hears about v2.1.0
	v, ok := c.Start().Wait(2 * time.Second)
	require.True(t, ok)
	require.Equal(t, Version{2, 1, 0}, v)
	require.EqualValues(t, 1, gh.hits.Load())

	// the next run, a minute later, neither asks nor nags
	clk.now = clk.now.Add(time.Minute)
	_, ok = c.Start().Wait(2 * time.Second)
	require.False(t, ok)
	require.EqualValues(t, 1, gh.hits.Load())

	// five hours on, still inside the interval
	clk.now = clk.now.Add(5 * time.Hour)
	_, ok = c.Start().Wait(2 * time.Second)
	require.False(t, ok)
	require.EqualValues(t, 1, gh.hits.Load())

	// past six hours it asks again, and reminds
	clk.now = clk.now.Add(2 * time.Hour)
	v, ok = c.Start().Wait(2 * time.Second)
	require.True(t, ok)
	require.Equal(t, Version{2, 1, 0}, v)
	require.EqualValues(t, 2, gh.hits.Load())
}

func TestLazyNothingNewer(t *testing.T) {
	for _, latest := range []string{"v2.0.0", "v1.9.0"} {
		gh := newFakeGitHub(t, latest, nil)
		c := newChecker(t, gh.source(), &clock{now: time.Now()})
		_, ok := c.Start().Wait(2 * time.Second)
		require.False(t, ok, latest)
		require.EqualValues(t, 1, gh.hits.Load(), latest)
	}
}

func TestLazyKnownReleaseShownWithoutWaiting(t *testing.T) {
	// The reminder comes from the state file, so it needs no network at all
	gh := newFakeGitHub(t, "v2.1.0", nil)
	clk := &clock{now: time.Now()}
	c := newChecker(t, gh.source(), clk)
	c.save(state{CheckedAt: clk.now, Latest: "v2.1.0"})
	v, ok := c.Start().Wait(0)
	require.True(t, ok)
	require.Equal(t, Version{2, 1, 0}, v)
	require.Zero(t, gh.hits.Load())
}

func TestLazyBrokenNetworkIsQuietAndNotRetriedEveryRun(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	clk := &clock{now: time.Now()}
	c := newChecker(t, &Source{API: srv.URL, Repo: "o/r", Client: srv.Client(), UserAgent: "test"}, clk)

	_, ok := c.Start().Wait(2 * time.Second)
	require.False(t, ok)
	clk.now = clk.now.Add(time.Minute)
	_, ok = c.Start().Wait(2 * time.Second)
	require.False(t, ok)
	require.EqualValues(t, 1, hits.Load(), "a failure counts as a try")
}

func TestLazyDoesNotWaitOnASlowServer(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	c := newChecker(t, &Source{API: srv.URL, Repo: "o/r", Client: srv.Client(), UserAgent: "test"}, &clock{now: time.Now()})

	start := time.Now()
	_, ok := c.Start().Wait(100 * time.Millisecond)
	require.False(t, ok)
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestLazyDamagedStateFile(t *testing.T) {
	gh := newFakeGitHub(t, "v2.1.0", nil)
	c := newChecker(t, gh.source(), &clock{now: time.Now()})
	require.NoError(t, os.MkdirAll(filepath.Dir(c.StatePath), 0o750))
	require.NoError(t, os.WriteFile(c.StatePath, []byte("{not json"), 0o600))
	_, ok := c.Start().Wait(2 * time.Second)
	require.True(t, ok, "starts over rather than failing")
}

func TestNilLazy(t *testing.T) {
	var l *Lazy
	_, ok := l.Wait(time.Second)
	require.False(t, ok)
}

// release makes the files of a fake release: an archive holding the given
// files, and the checksums for it
func release(t *testing.T, v Version, sumOverride string, entries map[string]string) map[string][]byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}))
		_, err := tw.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	sum := sha256.Sum256(buf.Bytes())
	hexSum := hex.EncodeToString(sum[:])
	if sumOverride != "" {
		hexSum = sumOverride
	}
	return map[string][]byte{
		archiveName(v):   buf.Bytes(),
		checksumsName(v): []byte(fmt.Sprintf("%s  %s\n%s  other-file.tar.gz\n", hexSum, archiveName(v), hexSum)),
	}
}

func installTarget(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), binaryName())
	require.NoError(t, os.WriteFile(exe, []byte("old"), 0o755))
	return exe
}

// leftovers is anything in the directory of exe other than exe
func leftovers(t *testing.T, exe string) []string {
	t.Helper()
	ents, err := os.ReadDir(filepath.Dir(exe))
	require.NoError(t, err)
	var out []string
	for _, e := range ents {
		if e.Name() != filepath.Base(exe) {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestInstall(t *testing.T) {
	v := Version{2, 1, 0}
	gh := newFakeGitHub(t, v.Tag(), release(t, v, "", map[string]string{
		"completions/taskpoet.bash": "not this",
		binaryName():                "new",
	}))
	exe := installTarget(t)
	require.NoError(t, os.Chmod(exe, 0o750))

	require.NoError(t, gh.source().Install(context.Background(), v, exe))

	got, err := os.ReadFile(exe)
	require.NoError(t, err)
	require.Equal(t, "new", string(got))
	st, err := os.Stat(exe)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o750), st.Mode().Perm(), "keeps the mode of what it replaces")
	require.Empty(t, leftovers(t, exe))
}

func TestInstallFailuresLeaveTheOldOneAlone(t *testing.T) {
	v := Version{2, 1, 0}
	good := map[string]string{binaryName(): "new"}
	tests := map[string]struct {
		files   map[string][]byte
		wantErr string
	}{
		"bad checksum": {
			files:   release(t, v, "0000000000000000000000000000000000000000000000000000000000000000", good),
			wantErr: "checksum",
		},
		"no binary in the archive": {
			files:   release(t, v, "", map[string]string{"README.md": "hi"}),
			wantErr: "no " + binaryName(),
		},
		"binary only in a subdirectory is still just its name": {
			files: release(t, v, "", map[string]string{"../../evil/" + binaryName(): "new"}),
		},
		"no such release": {
			files:   map[string][]byte{},
			wantErr: "404",
		},
		"platform not in the checksums": {
			files: map[string][]byte{
				checksumsName(v): []byte("abc  taskpoet-2.1.0_plan9_mips.tar.gz\n"),
			},
			wantErr: "no checksum",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gh := newFakeGitHub(t, v.Tag(), tt.files)
			exe := installTarget(t)
			err := gh.source().Install(context.Background(), v, exe)
			if tt.wantErr == "" {
				// not a failure at all, but it must not write anywhere else
				require.NoError(t, err)
				require.Empty(t, leftovers(t, exe))
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			got, rerr := os.ReadFile(exe)
			require.NoError(t, rerr)
			require.Equal(t, "old", string(got))
			require.Empty(t, leftovers(t, exe))
		})
	}
}

func TestManagedBy(t *testing.T) {
	require.Equal(t, "Homebrew", ManagedBy("/usr/local/Cellar/taskpoet/2.0.0/bin/taskpoet"))
	require.Equal(t, "Homebrew", ManagedBy("/home/linuxbrew/.linuxbrew/Cellar/taskpoet/2.0.0/bin/taskpoet"))
	require.Empty(t, ManagedBy("/usr/local/bin/taskpoet"))
	require.Empty(t, ManagedBy("/Users/me/bin/taskpoet"))
}

func TestChecksumFor(t *testing.T) {
	sums := []byte("ABCDEF  a.tar.gz\n123456  b.tar.gz\n")
	got, err := checksumFor(sums, "a.tar.gz")
	require.NoError(t, err)
	require.Equal(t, "abcdef", got)
	_, err = checksumFor(sums, "c.tar.gz")
	require.Error(t, err)
}
