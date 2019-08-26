// +build integration

package dynamodb

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

func TestDynamoDB_Resources(t *testing.T) {
	cfg := testConfig(t)

	table := "test-resources"
	done := createTestTable(t, cfg, table)
	defer done()

	registry := &resource.Registry{
		Types: map[string]reflect.Type{
			"foo": reflect.TypeOf(struct {
				Input  string `func:"input"`
				Output string `func:"output"`
			}{}),
		},
	}

	project := "testproject"
	ddb := New(cfg, table, registry)
	ctx := context.Background()

	resA := &resource.Resource{
		Type:   "foo",
		Name:   "a",
		Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("abc")}),
		Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("def")}),
	}
	resB := &resource.Resource{
		Type:    "foo",
		Name:    "b",
		Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("123")}),
		Output:  cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("456")}),
		Sources: []string{"x", "y", "z"},
		Deps:    []string{"foo", "bar"},
	}

	// Create
	if err := ddb.PutResource(ctx, project, resA); err != nil {
		t.Fatal(err)
	}
	if err := ddb.PutResource(ctx, project, resB); err != nil {
		t.Fatal(err)
	}

	got, err := ddb.ListResources(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	want := []*resource.Resource{
		resA,
		resB,
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}

	// Update
	update := &resource.Resource{
		Type:   "foo",
		Name:   "a", // Same name
		Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("ABC")}),
		Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("DEF")}),
	}
	if err := ddb.PutResource(ctx, project, update); err != nil {
		t.Fatal(err)
	}

	// Delete
	if err := ddb.DeleteResource(ctx, project, resB); err != nil {
		t.Fatal(err)
	}

	got, err = ddb.ListResources(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	want = []*resource.Resource{
		update, // a is updated
		// b is deleted
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestDynamoDB_DeleteResource_nonexisting(t *testing.T) {
	cfg := testConfig(t)

	table := "test-graphs"
	done := createTestTable(t, cfg, table)
	defer done()

	ddb := New(cfg, table, nil)
	ctx := context.Background()

	err := ddb.DeleteResource(ctx, "foo", &resource.Resource{Name: "bar"})
	if err == nil {
		t.Errorf("Want error when deleting non-existing resource")
	}
}

func TestDynamoDB_Graphs(t *testing.T) {
	cfg := testConfig(t)

	table := "test-graphs"
	done := createTestTable(t, cfg, table)
	defer done()

	registry := &resource.Registry{
		Types: map[string]reflect.Type{
			"person": reflect.TypeOf(struct {
				Name string `func:"input"`
				Age  int    `func:"input"`
			}{}),
		},
	}

	project := "testproject"
	ddb := New(cfg, table, registry)
	ctx := context.Background()

	g := &graph.Graph{
		Resources: map[string]*resource.Resource{
			"alice": {
				Name:    "alice",
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("alice"),
					"age":  cty.NumberIntVal(20),
				}),
			},
			"bob": {
				Name:    "bob",
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("bob"),
					"age":  cty.NumberIntVal(30),
				}),
				Deps: []string{"alice", "carol"},
			},
		},
		Dependencies: map[string][]graph.Dependency{
			"bob": {{
				Field: cty.GetAttrPath("friends"),
				Expression: graph.Expression{
					graph.ExprReference{
						Path: cty.
							GetAttrPath("alice").
							GetAttr("friends").
							Index(cty.NumberIntVal(0)),
					},
				},
			}},
		},
	}

	// Create
	if err := ddb.PutGraph(ctx, project, g); err != nil {
		t.Fatal(err)
	}

	got, err := ddb.GetGraph(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, g, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

var opts = []cmp.Option{
	cmpopts.EquateEmpty(),
	cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
	cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
	cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "Deps" || p.String() == "Sources" // String sets are not sorted
	}, cmpopts.SortSlices(func(a, b string) bool {
		return a < b
	})),
}

func testConfig(t *testing.T) aws.Config {
	accessKey := os.Getenv("TEST_DYNAMODB_ACCESS_KEY")
	secretKey := os.Getenv("TEST_DYNAMODB_SECRET_KEY")
	region := os.Getenv("TEST_DYNAMODB_REGION")
	endpoint := os.Getenv("TEST_DYNAMODB_ENDPOINT")

	if endpoint == "" {
		t.Skip("TEST_DYNAMODB_ENDPOINT not set")
	}

	cfg := defaults.Config()
	cfg.Region = region
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(endpoint)
	cfg.Credentials = aws.NewStaticCredentialsProvider(accessKey, secretKey, "")

	t.Logf("DynamoDB test endpoint: %s", endpoint)

	return cfg
}

func createTestTable(t *testing.T, cfg aws.Config, tableName string) func() {
	ddb := &DynamoDB{
		Client:    dynamodb.New(cfg),
		TableName: tableName,
	}

	t.Logf("Creating test table %q", tableName)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create test table
	if err := ddb.CreateTable(ctx, 1, 1); err != nil {
		t.Fatalf("Create resource table: %v", err)
	}

	cleanup := func() {
		// Delete table
		t.Logf("Deleting test table %q", tableName)
		cli := dynamodb.New(cfg)
		_, err := cli.DeleteTableRequest(&dynamodb.DeleteTableInput{
			TableName: aws.String(tableName),
		}).Send(context.Background())
		if err != nil {
			t.Fatalf("Delete DynamoDB table: %v", err)
		}
	}

	return cleanup
}
