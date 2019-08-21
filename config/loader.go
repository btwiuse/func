package config

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

type file struct {
	name  string
	bytes []byte
	body  *hclpack.Body
}

func (f *file) empty() bool {
	return len(f.body.ChildBlocks) == 0 && len(f.body.Attributes) == 0
}

// SourceCompressor is used for compressing the source files on disk to an
// archive that can be uploaded.
type SourceCompressor interface {
	// Compress compresses the given directory into w. The returned extension
	// is the extension for the file with leading dot (.tar.gz).
	Compress(w io.Writer, dir string) error
}

// A Loader loads configuration files from .hcl files on disk.
//
// If the Compressor is not set, the source files are not compressed and the
// source attribute is only removed from the output.
//
// The zero value is ready to load files.
type Loader struct {
	Compressor SourceCompressor

	files   map[string]*file
	sources map[string]*bytes.Buffer
}

// WriteDiagnostics writes diagnostics as a human readable string to w. It
// should only be used for diagnostics that originate from files loaded by
// Loader.
//
// If a TTY is attached, the output will be colorized and wrap at the terminal
// width. Otherwise, wrap will occur at 78 characters and output won't contain
// ANSI escape characters.
func (l *Loader) WriteDiagnostics(w io.Writer, diags hcl.Diagnostics) {
	files := make(map[string]*hcl.File, len(l.files))
	for name, f := range l.files {
		files[name] = &hcl.File{
			Bytes: f.bytes,
			Body:  f.body,
		}
	}
	cols, _, err := terminal.GetSize(0)
	if err != nil {
		cols = 78
	}
	color := terminal.IsTerminal(0)
	wr := hcl.NewDiagnosticTextWriter(w, files, uint(cols), color)
	if err := wr.WriteDiagnostics(diags); err != nil {
		fmt.Fprintln(w, err)
	}
}

// Root finds the root directory of a project. The returned string is the
// absolute path to the project on disk.
//
// The root directory is determined by the file .func/root existing. The
// contents of the file are not significant. If the given dir does not contain
// a project, parent directories are traversed until a project is found.
//
// An error is returned if the dir cannot be opened. An empty string is
// returned if no project root was found.
func (l *Loader) Root(dir string) (string, error) {
	// Check that dir itself exists
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	rootfile := filepath.Join(dir, ".func", "root")
	stat, err := os.Stat(rootfile)
	if err == nil && !stat.IsDir() {
		// Match
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	parent := filepath.Dir(dir)
	if parent == dir || parent[len(parent)-1] == filepath.Separator {
		return "", nil
	}

	return l.Root(parent)
}

// Load loads all the config files from the given root directory, traversing
// into sub directories.
//
// If resource blocks are encountered and they contain a source attribute, the
// source files from resource are collected and processed as described in the
// package documentation.
//
// If an empty .hcl file is encountered, it is not added.
func (l *Loader) Load(root string) (*hclpack.Body, hcl.Diagnostics) {
	var bodies []*hclpack.Body
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		if info.IsDir() {
			return nil
		}
		if !isConfigFile(path) {
			return nil
		}

		f, diags := l.loadFile(path)
		if diags.HasErrors() {
			return diags
		}

		if f.empty() {
			return nil
		}

		for i, b := range f.body.ChildBlocks {
			if b.Type == "resource" {
				block, diags := l.processResource(b, path)
				if diags.HasErrors() {
					return diags
				}
				f.body.ChildBlocks[i] = block
			}
		}

		bodies = append(bodies, f.body)
		return nil
	})
	if err != nil {
		if d, ok := err.(hcl.Diagnostics); ok {
			return nil, d
		}
		return nil, diagErr(err)
	}
	return mergeBodies(bodies), nil
}

// Source returns the compressed source for a given digest.
//
// The digests are encoded into the body returned from Load. When source files
// are needed for a given digest, the list of files can be returned with
// Source().
//
// The result is only valid if Load() has been executed without error.
func (l *Loader) Source(sha256 string) *bytes.Buffer {
	return l.sources[sha256]
}

func isConfigFile(filename string) bool {
	return filepath.Ext(filename) == ".hcl"
}

func (l *Loader) loadFile(filename string) (*file, hcl.Diagnostics) {
	if l.files == nil {
		l.files = make(map[string]*file)
	}
	if f, ok := l.files[filename]; ok {
		return f, nil
	}

	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, diagErr(err)
	}

	// Add placeholder file, so diagnostics can match the source if packing the
	// file fails.
	l.files[filename] = &file{bytes: src}

	body, diags := hclpack.PackNativeFile(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, diags
	}

	f := &file{
		name:  filename,
		bytes: src,
		body:  body,
	}
	l.files[filename] = f

	return f, nil
}

func (l *Loader) processResource(block hclpack.Block, filename string) (hclpack.Block, hcl.Diagnostics) {
	if srcAttr, ok := block.Body.Attributes["source"]; ok {
		var src string
		diags := gohcl.DecodeExpression(&srcAttr.Expr, nil, &src)
		if diags.HasErrors() {
			return hclpack.Block{}, diags
		}

		// Delete source attribute; no longer needed.
		delete(block.Body.Attributes, "source")

		dir := filepath.Dir(filename)
		dir = filepath.Join(dir, src)

		var buf bytes.Buffer
		sha := sha256.New()
		md5 := md5.New()

		w := io.MultiWriter(&buf, sha, md5)

		if err := l.Compressor.Compress(w, dir); err != nil {
			return hclpack.Block{}, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Could not create source archive: %v", err),
				Subject:  srcAttr.Expr.StartRange().Ptr(),
				Context:  srcAttr.Expr.Range().Ptr(),
			}}
		}

		key := hex.EncodeToString(sha.Sum(nil))

		if l.sources == nil {
			l.sources = make(map[string]*bytes.Buffer)
		}
		l.sources[key] = &buf

		srcInfo := SourceInfo{
			Len: buf.Len(),
			MD5: base64.StdEncoding.EncodeToString(md5.Sum(nil)),
			Key: key,
		}

		srcAttr.Expr = hclpack.Expression{
			Source:      []byte(`"` + srcInfo.EncodeToString() + `"`),
			SourceType:  hclpack.ExprLiteralJSON,
			Range_:      srcAttr.Expr.Range_,
			StartRange_: srcAttr.Expr.StartRange_,
		}
		block.Body.Attributes["source"] = srcAttr
	}
	return block, nil
}

// mergeBodies merges the contents of the given bodies.
//
// It behaves in a similar way to hcl.MergeBodies, except the *hclpack.Body
// struct type is returned instead of the hcl.Body interface.
//
// The missing range is arbitrarily set to the first file.
func mergeBodies(bodies []*hclpack.Body) *hclpack.Body {
	ret := &hclpack.Body{}
	for _, b := range bodies {
		for name, attr := range b.Attributes {
			if ret.Attributes == nil {
				ret.Attributes = make(map[string]hclpack.Attribute)
			}
			ret.Attributes[name] = attr
		}
		ret.ChildBlocks = append(ret.ChildBlocks, b.ChildBlocks...)
	}
	ret.MissingItemRange_ = bodies[0].MissingItemRange_
	return ret
}

// diagErr converts a native error to diagnostics
func diagErr(err error) hcl.Diagnostics {
	return hcl.Diagnostics{{Severity: hcl.DiagError, Summary: err.Error()}}
}
