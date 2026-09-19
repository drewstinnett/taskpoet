package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// Repo is where taskpoet is released
	Repo = "drewstinnett/taskpoet"

	maxAPIResponse = 1 << 20   // 1MiB
	maxChecksums   = 1 << 20   // 1MiB
	maxArchive     = 100 << 20 // 100MiB
	maxBinary      = 200 << 20 // 200MiB, once unpacked
)

// Source is the GitHub releases of a repository, and the file naming that
// goreleaser gives them, see .goreleaser.yaml
type Source struct {
	// API is where the releases are listed, and Download where their files are
	API, Download string
	Repo          string
	Client        *http.Client
	UserAgent     string
}

// NewSource is the real GitHub. current is only used to say who is asking.
func NewSource(current string) *Source {
	return &Source{
		API:       "https://api.github.com",
		Download:  "https://github.com",
		Repo:      Repo,
		Client:    &http.Client{},
		UserAgent: "taskpoet/" + current,
	}
}

func (s *Source) get(ctx context.Context, url, accept string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.UserAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("GET %s: more than %d bytes", url, limit)
	}
	return body, nil
}

// Latest is the newest published release. GitHub leaves out drafts and
// pre-releases.
func (s *Source) Latest(ctx context.Context) (Version, error) {
	body, err := s.get(ctx, s.API+"/repos/"+s.Repo+"/releases/latest", "application/vnd.github+json", maxAPIResponse)
	if err != nil {
		return Version{}, err
	}
	var rel struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &rel); err != nil {
		return Version{}, fmt.Errorf("reading the latest release: %w", err)
	}
	return ParseVersion(rel.Tag)
}

func (s *Source) downloadURL(v Version, file string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s/%s", s.Download, s.Repo, v.Tag(), file)
}

func archiveName(v Version) string {
	return fmt.Sprintf("taskpoet-%s_%s_%s.tar.gz", v, runtime.GOOS, runtime.GOARCH)
}

func checksumsName(v Version) string {
	return fmt.Sprintf("taskpoet-%s_SHA256SUMS", v)
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "taskpoet.exe"
	}
	return "taskpoet"
}

// Install replaces the executable at exe with release v for this platform.
// The download is checked against the checksums published with the release,
// which catches a damaged download. It can't tell you the release itself is
// genuine, you are trusting GitHub and the repository for that.
//
// exe is only touched once everything has been downloaded and checked, and it
// is swapped in a single rename, so a failure leaves the old one working.
func (s *Source) Install(ctx context.Context, v Version, exe string) error {
	name := archiveName(v)
	sums, err := s.get(ctx, s.downloadURL(v, checksumsName(v)), "", maxChecksums)
	if err != nil {
		return err
	}
	want, err := checksumFor(sums, name)
	if err != nil {
		return err
	}
	archive, err := s.get(ctx, s.downloadURL(v, name), "", maxArchive)
	if err != nil {
		return err
	}
	if sum := sha256.Sum256(archive); hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("%s does not match its checksum, it was probably damaged in the download; try again", name)
	}
	tmp, err := unpackBinary(archive, filepath.Dir(exe), fileMode(exe))
	if err != nil {
		return err
	}
	if err := replaceExecutable(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// checksumFor finds a file in a SHA256SUMS listing, "<hex>  <file>" per line
func checksumFor(sums []byte, file string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[1] == file {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("no checksum for %s in the release, there may be no build of it for this platform", file)
}

func fileMode(p string) fs.FileMode {
	if st, err := os.Stat(p); err == nil {
		return st.Mode().Perm()
	}
	return 0o755
}

// unpackBinary writes the taskpoet executable in a release archive to a new
// file in dir, and returns its path
func unpackBinary(archive []byte, dir string, mode fs.FileMode) (string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return "", fmt.Errorf("reading the release archive: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("no %s in the release archive", binaryName())
		}
		if err != nil {
			return "", fmt.Errorf("reading the release archive: %w", err)
		}
		// Only ever look at the name, never build a path from the archive's
		if hdr.Typeflag != tar.TypeReg || path.Base(hdr.Name) != binaryName() {
			continue
		}
		return writeTemp(io.LimitReader(tr, maxBinary+1), dir, mode)
	}
}

func writeTemp(r io.Reader, dir string, mode fs.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, ".taskpoet-update-*")
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxBinary {
		err = fmt.Errorf("the executable is larger than %d bytes", maxBinary)
	}
	if err == nil {
		err = os.Chmod(f.Name(), mode)
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// replaceExecutable moves tmp over exe. Everywhere but Windows that is one
// atomic rename, and it works while exe is running. Windows won't overwrite a
// running program, but lets it be renamed out of the way.
func replaceExecutable(tmp, exe string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(tmp, exe)
	}
	old := exe + ".old"
	_ = os.Remove(old) // left by an earlier update, gone once that one exited
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(old, exe)
		return err
	}
	_ = os.Remove(old) // fails while we run, the next update gets it
	return nil
}

// Executable is the path of the running program with symlinks followed, so an
// update replaces the file itself and not a link to it
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// ManagedBy names the package manager that owns the executable at exe, or is
// empty. Those should be updated with the package manager, not behind its back.
func ManagedBy(exe string) string {
	if strings.Contains(filepath.ToSlash(exe), "/Cellar/") {
		return "Homebrew"
	}
	return ""
}

// IsPermission is true if err is about not being allowed to write
func IsPermission(err error) bool {
	return errors.Is(err, fs.ErrPermission)
}
