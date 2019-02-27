package apigatewaypatch

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestResolve(t *testing.T) {
	strptr := func(str string) *string { return &str }
	i64ptr := func(v int64) *int64 { return &v }

	type s struct {
		List []string
	}

	type def struct {
		Str       string
		StrPtr    *string
		List      []string
		ListPtr   *[]string
		StructPtr *s
		Int64     *int64
	}

	fields := []Field{
		{Name: "Str", Path: "/str"},
		{Name: "StrPtr", Path: "/strptr"},
		{Name: "List", Path: "/list"},
		{Name: "ListPtr", Path: "/listptr"},
		{Name: "StructPtr.List", Path: "/structptr/list"},
		{Name: "Int64", Path: "/i64"},
	}

	tests := []struct {
		name    string
		prev    interface{}
		next    interface{}
		fields  []Field
		want    []apigateway.PatchOperation
		wantErr bool
	}{
		{
			name:   "Empty",
			prev:   &def{},
			next:   &def{},
			fields: fields,
			want:   nil,
		},
		{
			name:   "Noop",
			prev:   &def{Str: "hello", StrPtr: strptr("world")},
			next:   &def{Str: "hello", StrPtr: strptr("world")},
			fields: fields,
			want:   nil,
		},
		{
			name:    "FieldNotFound",
			prev:    &def{Str: "hello"},
			next:    &def{Str: "hello"},
			fields:  []Field{{Name: "notfound"}},
			wantErr: true,
		},
		{
			name:   "Set",
			prev:   &def{},
			next:   &def{Str: "hi"},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpReplace, Path: strptr("/str"), Value: strptr("hi")},
			},
		},
		{
			name:   "Clear",
			prev:   &def{Str: "hi", StrPtr: strptr("there")},
			next:   &def{},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpReplace, Path: strptr("/str"), Value: nil},
				{Op: apigateway.OpReplace, Path: strptr("/strptr"), Value: nil},
			},
		},
		{
			name:   "UpdateString",
			prev:   &def{Str: "hello", StrPtr: strptr("world")},
			next:   &def{Str: "hi", StrPtr: strptr("world")},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpReplace, Path: strptr("/str"), Value: strptr("hi")},
			},
		},
		{
			name:   "UpdateStringPointer",
			prev:   &def{Str: "hello", StrPtr: strptr("world")},
			next:   &def{Str: "hello", StrPtr: strptr("world!")},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpReplace, Path: strptr("/strptr"), Value: strptr("world!")},
			},
		},
		{
			name:   "UpdateListRemove",
			prev:   &def{List: []string{"a", "b"}},
			next:   &def{List: []string{"a"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpRemove, Path: strptr("/list/b")},
			},
		},
		{
			name:   "UpdateListAdd",
			prev:   &def{List: []string{"a", "b"}},
			next:   &def{List: []string{"a", "b", "c"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpAdd, Path: strptr("/list/c")},
			},
		},
		{
			name:   "UpdateListChange",
			prev:   &def{List: []string{"a", "b"}},
			next:   &def{List: []string{"a", "c"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpRemove, Path: strptr("/list/b")},
				{Op: apigateway.OpAdd, Path: strptr("/list/c")},
			},
		},
		{
			name:   "UpdateListRemoveHead",
			prev:   &def{List: []string{"a", "b"}},
			next:   &def{List: []string{"b"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpRemove, Path: strptr("/list/a")},
			},
		},
		{
			name:   "UpdateListInsertHead",
			prev:   &def{List: []string{"b"}},
			next:   &def{List: []string{"a", "b"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpAdd, Path: strptr("/list/a")},
			},
		},
		{
			name:   "ListDiffOrder",
			prev:   &def{List: []string{"a", "b"}},
			next:   &def{List: []string{"b", "a"}},
			fields: fields,
			want:   nil,
		},
		{
			name:   "UpdateListPointer",
			prev:   &def{ListPtr: &[]string{"a", "b"}},
			next:   &def{ListPtr: &[]string{"a", "c"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpRemove, Path: strptr("/listptr/b")},
				{Op: apigateway.OpAdd, Path: strptr("/listptr/c")},
			},
		},
		{
			name:   "UpdateListPointerNilAdd",
			prev:   &def{},
			next:   &def{ListPtr: &[]string{"a"}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpAdd, Path: strptr("/listptr/a")},
			},
		},
		{
			name:   "UpdateListPointerNilRemove",
			prev:   &def{ListPtr: &[]string{"a"}},
			next:   &def{},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpRemove, Path: strptr("/listptr/a")},
			},
		},
		{
			name:   "StructPointerList",
			prev:   &def{StructPtr: &s{List: []string{"a", "b"}}},
			next:   &def{StructPtr: &s{List: []string{"a", "x", "c"}}},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpRemove, Path: strptr("/structptr/list/b")},
				{Op: apigateway.OpAdd, Path: strptr("/structptr/list/c")},
				{Op: apigateway.OpAdd, Path: strptr("/structptr/list/x")},
			},
		},
		{
			name:   "Int64Ptr",
			prev:   &def{Int64: i64ptr(111)},
			next:   &def{Int64: i64ptr(999)},
			fields: fields,
			want: []apigateway.PatchOperation{
				{Op: apigateway.OpReplace, Path: strptr("/i64"), Value: strptr("999")},
			},
		},
		{
			name: "Modifier",
			prev: &def{Str: "hello", StrPtr: strptr("foo")},
			next: &def{Str: "world", StrPtr: strptr("bar")},
			fields: []Field{
				{Name: "StrPtr", Path: "/strptr"}, // no modifier
				{Name: "Str", Path: "/str", Modifier: func(ops []apigateway.PatchOperation) []apigateway.PatchOperation {
					want := []apigateway.PatchOperation{
						{Op: apigateway.OpReplace, Path: strptr("/str"), Value: strptr("world")},
					}
					if diff := cmp.Diff(ops, want, cmpopts.IgnoreUnexported(apigateway.PatchOperation{})); diff != "" {
						t.Logf("PatchOperation (-got, +want)\n%s", diff)
						// Cause test to fail, don't have *testing.T here.
						return nil
					}
					op := ops[0]
					return []apigateway.PatchOperation{
						{Op: apigateway.OpRemove, Path: op.Path},
						{Op: apigateway.OpAdd, Path: op.Path, Value: op.Value},
					}
				}},
			},
			want: []apigateway.PatchOperation{
				// Normal output for field without modifier
				{Op: apigateway.OpReplace, Path: strptr("/strptr"), Value: strptr("bar")},
				// Modified output
				{Op: apigateway.OpRemove, Path: strptr("/str")},
				{Op: apigateway.OpAdd, Path: strptr("/str"), Value: strptr("world")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.prev, tt.next, tt.fields...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Resolve() error = %v, want error = %t", err, tt.wantErr)
			}

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(apigateway.PatchOperation{}),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("PatchOperation (-got, +want)\n%s", diff)
			}
		})
	}
}
