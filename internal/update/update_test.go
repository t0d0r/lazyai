package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCompare(t *testing.T) {
	for s, want := range map[string]Version{"v0.3.0": {0, 3, 0}, "0.3.0": {0, 3, 0}, "v1.12.7": {1, 12, 7}} {
		if v, ok := Parse(s); !ok || v != want {
			t.Errorf("Parse(%q) = %v, %v", s, v, ok)
		}
	}
	for _, s := range []string{"dev", "v0.3.0-2-gabc1234", "v0.3.0-dirty", "abc1234", "v0.3", ""} {
		if _, ok := Parse(s); ok {
			t.Errorf("Parse(%q) accepted a non-release version", s)
		}
	}
	cases := []struct {
		a, b Version
		want int
	}{
		{Version{0, 2, 1}, Version{0, 3, 0}, -1},
		{Version{0, 10, 0}, Version{0, 9, 9}, 1},
		{Version{1, 0, 0}, Version{0, 99, 99}, 1},
		{Version{0, 3, 0}, Version{0, 3, 0}, 0},
	}
	for _, c := range cases {
		if got := c.a.Compare(c.b); got != c.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/t0d0r/lazyai/releases/latest" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/t0d0r/lazyai/releases/tag/v0.4.1", http.StatusFound)
	}))
	defer srv.Close()
	c := New()
	c.Base = srv.URL
	v, err := c.Latest()
	if err != nil || v != (Version{0, 4, 1}) {
		t.Fatalf("Latest() = %v, %v", v, err)
	}
	c.Repo = "t0d0r/missing"
	if _, err := c.Latest(); err == nil {
		t.Error("Latest() of a repository without releases succeeded")
	}
}

func TestVerify(t *testing.T) {
	data := []byte("binary")
	sum := sha256.Sum256(data)
	sums := []byte(hex.EncodeToString(sum[:]) + "  a.tar.gz\n" + strings.Repeat("0", 64) + " *b.deb\n")
	if err := Verify(sums, "a.tar.gz", data); err != nil {
		t.Errorf("matching sum: %v", err)
	}
	if err := Verify(sums, "b.deb", data); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("wrong sum: %v", err)
	}
	if err := Verify(sums, "c.tar.gz", data); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Errorf("missing entry: %v", err)
	}
}

type entry struct {
	name string
	typ  byte
	body string
}

func tarball(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: 0o755, Size: int64(len(e.body))}
		if e.typ == tar.TypeSymlink {
			h.Size, h.Linkname = 0, "/etc/passwd"
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if e.typ == tar.TypeReg {
			tw.Write([]byte(e.body))
		}
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

func TestExtractBinary(t *testing.T) {
	dir := "lazyai-v0.4.1-linux-amd64"
	tgz := tarball(t,
		entry{dir + "/README.md", tar.TypeReg, "readme"},
		entry{dir + "/lazyai", tar.TypeReg, "new binary"},
	)
	bin, err := ExtractBinary(tgz, dir)
	if err != nil || string(bin) != "new binary" {
		t.Fatalf("ExtractBinary = %q, %v", bin, err)
	}
	if _, err := ExtractBinary(tgz, "lazyai-v0.4.1-darwin-arm64"); err == nil {
		t.Error("extracted from the wrong directory")
	}
	link := tarball(t, entry{dir + "/lazyai", tar.TypeSymlink, ""})
	if _, err := ExtractBinary(link, dir); err == nil {
		t.Error("extracted a symlink")
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "lazyai")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Replace(exe, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	fi, _ := os.Stat(exe)
	if string(got) != "new" || fi.Mode().Perm() != 0o755 {
		t.Errorf("after Replace: %q, mode %v", got, fi.Mode())
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".lazyai-update-*")); len(left) > 0 {
		t.Errorf("temporary files left: %v", left)
	}
	if os.Geteuid() != 0 {
		os.Chmod(dir, 0o555)
		defer os.Chmod(dir, 0o755)
		if err := Replace(exe, []byte("x")); err == nil || !strings.Contains(err.Error(), "cannot write") {
			t.Errorf("read-only directory: %v", err)
		}
	}
}

func TestDetect(t *testing.T) {
	owns := func(p string) bool { return p == "/usr/bin/lazyai" }
	cases := []struct {
		exe, goos string
		want      Method
	}{
		{"/opt/homebrew/Cellar/lazyai/0.3.0/bin/lazyai", "darwin", Homebrew},
		{"/home/linuxbrew/.linuxbrew/Cellar/lazyai/0.3.0/bin/lazyai", "linux", Homebrew},
		{"/usr/bin/lazyai", "linux", Deb},
		{"/usr/bin/lazyai", "darwin", Binary},
		{"/home/u/.local/bin/lazyai", "linux", Binary},
	}
	for _, c := range cases {
		if got := Detect(c.exe, c.goos, owns); got != c.want {
			t.Errorf("Detect(%q, %s) = %s, want %s", c.exe, c.goos, got, c.want)
		}
	}
}

// TestInstallBinary runs the whole binary update against a local server
// that serves a release like GitHub's.
func TestInstallBinary(t *testing.T) {
	v := Version{0, 4, 1}
	name := TarballName(v, "linux", "arm64")
	tgz := tarball(t, entry{strings.TrimSuffix(name, ".tar.gz") + "/lazyai", tar.TypeReg, "v0.4.1 binary"})
	sum := sha256.Sum256(tgz)
	sums := hex.EncodeToString(sum[:]) + "  " + name + "\n"
	files := map[string][]byte{"SHA256SUMS": []byte(sums), name: tgz}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := files[strings.TrimPrefix(r.URL.Path, "/t0d0r/lazyai/releases/download/v0.4.1/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	defer srv.Close()
	c := New()
	c.Base = srv.URL

	exe := filepath.Join(t.TempDir(), "lazyai")
	os.WriteFile(exe, []byte("old"), 0o755)
	if err := c.installBinary(v, exe, "linux", "arm64"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "v0.4.1 binary" {
		t.Errorf("binary after update: %q", got)
	}

	files[name] = append([]byte(nil), tgz[:len(tgz)-1]...) // truncated download
	os.WriteFile(exe, []byte("old"), 0o755)
	if err := c.installBinary(v, exe, "linux", "arm64"); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Errorf("truncated download: %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("binary changed after a failed check: %q", got)
	}
	if err := c.installBinary(v, exe, "plan9", "386"); err == nil {
		t.Error("installed for a platform without a release file")
	}
}
