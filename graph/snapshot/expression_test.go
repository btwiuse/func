package snapshot_test

import (
	"testing"

	"github.com/func/func/graph"
	"github.com/func/func/graph/snapshot"
)

func TestExpr_Eval(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		data    map[graph.Field]interface{}
		want    string
		wantErr bool
	}{
		{
			"Empty",
			"",
			nil,
			"",
			false,
		},
		{
			"NoData",
			"hello world",
			nil,
			"hello world",
			false,
		},
		{
			"WithData",
			"> ${foo.bar.baz} ${bar.baz.qux}!",
			map[graph.Field]interface{}{
				{Type: "foo", Name: "bar", Field: "baz"}: "hi",
				{Type: "bar", Name: "baz", Field: "qux"}: "there",
			},
			"> hi there!",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			templ := snapshot.Expr(tt.input)
			err := templ.Eval(tt.data, &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Eval() error = %v, wantErr = %t", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Eval() got = %q, want = %q", got, tt.want)
			}
		})
	}
}

func TestExpr_Eval_notPointer(t *testing.T) {
	var str string
	templ := snapshot.Expr("abc")
	err := templ.Eval(nil, str) // must be pointer
	if err == nil {
		t.Errorf("Error is nil")
	}
}

func TestExpr_Eval_notString(t *testing.T) {
	var number int
	templ := snapshot.Expr("abc")
	err := templ.Eval(nil, &number) // must be pointer to string
	if err == nil {
		t.Errorf("Error is nil")
	}
}
