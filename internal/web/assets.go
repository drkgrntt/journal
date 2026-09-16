package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"time"
)

// AssetVersion is a hash of every embedded asset's path+contents, computed
// once when the binary starts - any changed CSS/JS file (i.e. any new
// deploy, since these are compiled in via go:embed) produces a different
// value. Appended as a query string to asset URLs so browsers treat a
// changed file as a new URL instead of serving a stale cached copy, which
// in turn lets /assets be served with a long, aggressive Cache-Control
// (see routes.go) without risking staleness after a release.
var AssetVersion = computeAssetVersion()

func computeAssetVersion() string {
	h := sha256.New()
	_ = fs.WalkDir(Files, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, readErr := fs.ReadFile(Files, path)
		if readErr != nil {
			return readErr
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:10]
}

// AssetURL appends the current build's cache-busting version to a
// top-level asset path referenced directly from a <link>/<script> tag
// (base.templ). CSS `@import url(...)` and JS `import`/`from` specifiers
// referencing other local .css/.js files get the same version appended to
// them too (see versionedAssetFS below) - without that, busting only the
// entry tag would leave everything it pulls in (output.css's own chain of
// @imports, index.js's own chain of module imports) cached under their old,
// un-versioned URLs.
func AssetURL(path string) string {
	return path + "?v=" + AssetVersion
}

var (
	cssImportRe = regexp.MustCompile(`@import url\("(/assets/[^"]+\.css)"\)`)
	jsImportRe  = regexp.MustCompile(`\b(import|from)(\s+)"([^"]+\.js)"`)
)

// versionedAssetFS rewrites the .css/.js files it serves so their own
// internal references to other local .css/.js files carry AssetVersion too,
// then wraps the result as an http.File reporting the rewritten size -
// everything else (fonts, images, non-css/js files) passes through
// untouched.
type versionedAssetFS struct {
	http.FileSystem
}

// NewVersionedAssetFS wraps an asset http.FileSystem so every .css/.js file
// it serves has AssetVersion appended to its own @import/import specifiers
// (see versionedAssetFS.Open) - used by routes.go to wrap web.Files before
// handing it to the /assets filesystem middleware.
func NewVersionedAssetFS(inner http.FileSystem) http.FileSystem {
	return versionedAssetFS{FileSystem: inner}
}

func (v versionedAssetFS) Open(name string) (http.File, error) {
	f, err := v.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}

	if !hasSuffixAny(name, ".css", ".js") {
		return f, nil
	}

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return f, nil
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	switch {
	case hasSuffixAny(name, ".css"):
		b = cssImportRe.ReplaceAll(b, []byte(`@import url("$1?v=`+AssetVersion+`")`))
	case hasSuffixAny(name, ".js"):
		b = jsImportRe.ReplaceAll(b, []byte(`$1$2"$3?v=`+AssetVersion+`"`))
	}

	return &versionedFile{Reader: bytes.NewReader(b), info: versionedFileInfo{FileInfo: info, size: int64(len(b))}}, nil
}

func hasSuffixAny(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if len(s) >= len(suf) && s[len(s)-len(suf):] == suf {
			return true
		}
	}
	return false
}

type versionedFile struct {
	*bytes.Reader
	info versionedFileInfo
}

func (f *versionedFile) Close() error                       { return nil }
func (f *versionedFile) Stat() (fs.FileInfo, error)         { return f.info, nil }
func (f *versionedFile) Readdir(int) ([]fs.FileInfo, error) { return nil, nil }

type versionedFileInfo struct {
	fs.FileInfo
	size int64
}

func (i versionedFileInfo) Size() int64        { return i.size }
func (i versionedFileInfo) ModTime() time.Time { return i.FileInfo.ModTime() }
