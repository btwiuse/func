package structdoc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strings"

	"github.com/fatih/structtag"
	"github.com/pkg/errors"
)

// A Struct represents a parsed struct.
type Struct struct {
	Doc     string
	Comment string
	Fields  []Field
}

// A Field represents a parsed struct field.
type Field struct {
	Doc     string
	Comment string
	Name    string
	Pointer bool
	Type    string
	Tags    []Tag
}

// A Tag is a struct tag set on a struct field.
type Tag struct {
	Key     string
	Name    string
	Options []string
}

// Parse parses go source code from the given reader, looking for a struct type
// definition with a given name.
func Parse(r io.Reader, typename string) (*Struct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", r, parser.ParseComments)
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}

	for _, d := range f.Decls {
		decl, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range decl.Specs {
			spec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if spec.Name.Name != typename {
				continue
			}
			strct, ok := spec.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("type %q is a %T, want struct", typename, spec.Type)
			}

			fields, err := parseFields(strct)
			if err != nil {
				return nil, errors.Wrap(err, "parse fields")
			}

			return &Struct{
				Doc:     strings.TrimSpace(decl.Doc.Text()),
				Comment: strings.TrimSpace(spec.Comment.Text()),
				Fields:  fields,
			}, nil
		}
	}
	return nil, errors.Errorf("%s not found", typename)
}

func parseFields(t *ast.StructType) ([]Field, error) {
	var fields []Field // nolint: prealloc

	for _, f := range t.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		name := f.Names[0].Name
		tags, err := parseTags(f)
		if err != nil {
			return nil, errors.Wrapf(err, "parse %s struct tags", name)
		}
		out := Field{
			Doc:     strings.TrimSpace(f.Doc.Text()),
			Comment: strings.TrimSpace(f.Comment.Text()),
			Name:    name,
			Tags:    tags,
		}
		setType(f.Type, &out)
		fields = append(fields, out)
	}

	return fields, nil
}

func setType(expr ast.Expr, field *Field) {
	switch e := expr.(type) {
	case *ast.StarExpr:
		field.Pointer = true
		setType(e.X, field)
	case *ast.Ident:
		field.Type = e.Name
	case *ast.SelectorExpr:
		setType(e.X, field)
	case *ast.StructType:
		field.Type = "struct"
	case *ast.MapType:
		field.Type = "map"
	case *ast.ArrayType:
		field.Type = "array"
	default:
		panic(fmt.Sprintf("Unknown type %T", expr))
	}
}

func parseTags(field *ast.Field) ([]Tag, error) {
	if field.Tag == nil {
		return nil, nil
	}
	tagstr := strings.Replace(field.Tag.Value, "`", "", -1)
	tt, err := structtag.Parse(tagstr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse struct tag %s", tagstr)
	}
	tags := make([]Tag, tt.Len())
	for i, t := range tt.Tags() {
		tags[i] = Tag(*t)
	}
	return tags, nil
}
