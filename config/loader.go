package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/pkg/errors"
)

type file struct {
	name  string
	bytes []byte
	body  *hclpack.Body
}

func (f *file) empty() bool {
	return len(f.body.ChildBlocks) == 0 && len(f.body.Attributes) == 0
}

// A Loader loads configuration files from .hcl files on disk.
//
// The zero value is ready to load files.
type Loader struct {
	files   map[string]*file
	sources map[string][]string
}

// Root finds the root directory of a project.
//
// The root directory is determined by a directory containing a config file
// with a project definition.
//
// If the given dir does not contain a project, parent directories are
// traversed until a project is found. If no parent directory contains a
// project, ErrProjectNotFound is returned.
//
// Root will do the minimum necessary work to find the project. This means the
// directory may contain multiple projects, even if that is not allowed.
func (l *Loader) Root(dir string) (string, hcl.Diagnostics) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", diagErr(err)
	}

	for _, f := range files {
		if !isConfigFile(f.Name()) {
			continue
		}
		filename := filepath.Join(dir, f.Name())
		f, diags := l.loadFile(filename)
		if diags.HasErrors() {
			return "", diags
		}

		for _, block := range f.body.ChildBlocks {
			if block.Type == "project" {
				return dir, nil
			}
		}
	}

	parent := filepath.Dir(dir)
	if parent == dir || parent[len(parent)-1] == filepath.Separator {
		return "", hcl.Diagnostics{{Severity: hcl.DiagError, Summary: "Project not found"}}
	}

	return l.Root(parent)
}

// Load loads all the config files from the given root directory, traversing
// into sub directories.
//
// If resource blocks are encountered and they contain a source attribute, the
// source files from resource are collected and hashed. The source attribute is
// replaced with a digest containing the hash digest.
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

		for _, b := range f.body.ChildBlocks {
			if b.Type == "resource" {
				diags := l.processResource(&b, path)
				if diags.HasErrors() {
					return diags
				}
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

// PrintDiagnostics prints diagnostics to the given writer.
//
// If a TTY is attached, the output will be colorized and wrap at the terminal
// width. Otherwise, wrap will occur at 78 characters and output won't contain
// ANSI escape characters.
func (l *Loader) PrintDiagnostics(w io.Writer, diags hcl.Diagnostics) {
	files := l.Files()
	cols, _, err := terminal.GetSize(0)
	if err != nil {
		cols = 78
	}
	color := terminal.IsTerminal(0)
	wr := hcl.NewDiagnosticTextWriter(w, files, uint(cols), color)
	if err := wr.WriteDiagnostics(diags); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if diags.HasErrors() {
			os.Exit(1)
		}
	}
}

// Files returns the configuration files that were loaded.
//
// The resulting map can be passed as files to hcl.NewDiagnosticTextWriter for
// matching the diagnostics to original source files.
//
// The result is only valid if Load() has been executed without error.
func (l *Loader) Files() map[string]*hcl.File {
	list := make(map[string]*hcl.File, len(l.files))
	for name, f := range l.files {
		list[name] = &hcl.File{
			Bytes: f.bytes,
			Body:  f.body,
		}
	}
	return list
}

// Source returns the source files for a given digest.
//
// The digests are encoded into the body returned from Load. When source files
// are needed for a given digest, the list of files can be returned with
// Source().
//
// The result is only valid if Load() has been executed without error.
func (l *Loader) Source(digest string) []string {
	return l.sources[digest]
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

func (l *Loader) processResource(block *hclpack.Block, filename string) hcl.Diagnostics {
	if srcAttr, ok := block.Body.Attributes["source"]; ok {
		var src string
		diags := gohcl.DecodeExpression(&srcAttr.Expr, nil, &src)
		if diags.HasErrors() {
			return diags
		}

		dir := filepath.Dir(filename)
		dir = filepath.Join(dir, src)
		files, err := collectSource(dir, []string{filename})
		if err != nil {
			return hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Could not collect source files: %v", err),
					Subject:  srcAttr.Expr.StartRange().Ptr(),
					Context:  srcAttr.Expr.Range().Ptr(),
				},
			}
		}
		digest, err := hash(files)
		if err != nil {
			return hcl.Diagnostics{
				{Severity: hcl.DiagError, Summary: err.Error(), Context: &srcAttr.Range},
			}
		}

		if l.sources == nil {
			l.sources = make(map[string][]string)
		}
		l.sources[digest] = files

		// Add hash digest as attribute
		// Repurpose the range from source so it at least matches this resource
		// and points to the source, in case there's an error.
		block.Body.Attributes["digest"] = hclpack.Attribute{
			Expr: hclpack.Expression{
				Source:      []byte(`"` + digest + `"`),
				SourceType:  hclpack.ExprLiteralJSON,
				Range_:      srcAttr.Expr.Range_,
				StartRange_: srcAttr.Expr.StartRange_,
			},
			Range:     srcAttr.Range,
			NameRange: srcAttr.NameRange,
		}

		// Delete source attribute; no longer needed.
		delete(block.Body.Attributes, "source")
	}
	return nil
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

// collectSource returns all files in the given directory, except files that
// are set in exclude.
//
// The files are sorted in lexicographical order.
func collectSource(dir string, exclude []string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		for _, ex := range exclude {
			if ex == path {
				return filepath.SkipDir
			}
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return files, nil
}

// hash computes a hex encoded sha256 hash of the contents of the given files.
func hash(files []string) (string, error) {
	sha := sha256.New()
	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			return "", errors.WithStack(err)
		}
		_, err = io.Copy(sha, f)
		if err != nil {
			f.Close()
			return "", errors.WithStack(err)
		}
		f.Close()
	}
	return hex.EncodeToString(sha.Sum(nil)), nil
}

// diagErr converts a native error to diagnostics
func diagErr(err error) hcl.Diagnostics {
	return hcl.Diagnostics{{Severity: hcl.DiagError, Summary: err.Error()}}
}
