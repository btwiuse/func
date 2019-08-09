package ctyext_test

import (
	"testing"

	"github.com/func/func/ctyext"
	"github.com/zclconf/go-cty/cty"
)

func TestApplyTypePath(t *testing.T) {
	tests := []struct {
		name    string
		input   cty.Type
		path    cty.Path
		want    cty.Type
		wantErr string
	}{
		{
			name: "FieldFromObject",
			input: cty.Object(map[string]cty.Type{
				"foo": cty.Object(map[string]cty.Type{
					"bar": cty.String,
				}),
			}),
			path: cty.GetAttrPath("foo").GetAttr("bar"),
			want: cty.String,
		},
		{
			name:  "FieldFromMap",
			input: cty.Map(cty.String),
			path:  cty.GetAttrPath("foo"),
			want:  cty.String,
		},
		{
			name:  "ListIndex",
			input: cty.List(cty.String),
			path:  cty.IndexPath(cty.NumberIntVal(3)),
			want:  cty.String,
		},
		{
			name:  "ListMapIndex",
			input: cty.List(cty.Map(cty.Number)),
			path:  cty.IndexPath(cty.NumberIntVal(2)).GetAttr("val"),
			want:  cty.Number,
		},
		{
			name: "ObjectFieldNotFound",
			input: cty.Object(map[string]cty.Type{
				"foo": cty.String,
			}),
			path:    cty.GetAttrPath("bar"),
			wantErr: "no attribute named \"bar\"",
		},
		{
			name: "ObjectFieldNotFoundDeep",
			input: cty.Object(map[string]cty.Type{
				"foo": cty.Object(map[string]cty.Type{
					"bar": cty.Object(map[string]cty.Type{
						"baz": cty.Object(map[string]cty.Type{
							"qux": cty.String,
						}),
					}),
				}),
			}),
			path:    cty.GetAttrPath("foo").GetAttr("bar").GetAttr("abc"),
			wantErr: "no attribute named \"abc\" in foo.bar",
		},
		{
			name:    "AttrInMapString",
			input:   cty.Map(cty.String),
			path:    cty.GetAttrPath("foo").GetAttr("wrong"),
			wantErr: "cannot access nested type \"wrong\", foo is a string",
		},
		{
			name:    "AttrInString",
			input:   cty.String,
			path:    cty.GetAttrPath("foo"),
			wantErr: "cannot access nested type \"foo\" in string",
		},
		{
			name: "IndexInObject",
			input: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			path:    cty.GetAttrPath("foo").Index(cty.NumberIntVal(2)).Index(cty.NumberIntVal(1)),
			wantErr: "cannot access indexed type from string in foo[2]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ctyext.ApplyTypePath(tt.input, tt.path)
			if err != nil {
				if tt.wantErr == "" {
					t.Fatalf("Unexpected error %v", err)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("ApplyPath()\nGot err  = %v\nWant err = %s", err, tt.wantErr)
				}
				return
			}
			if tt.wantErr != "" {
				t.Fatalf("Got <nil> error, want error: %s", tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ApplyPath()\nGot  %s\nwant %s", got.GoString(), tt.want.GoString())
			}
		})
	}
}
