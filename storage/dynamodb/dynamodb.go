package dynamodb

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/schema"
	"github.com/func/func/storage/dynamodb/internal/attr"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// The Registry returns types for unmarshalling resource inputs/outputs.
type Registry interface {
	Type(typename string) reflect.Type
}

// DynamoDB stores data in AWS DynamoDB
type DynamoDB struct {
	Client    dynamodbiface.ClientAPI
	TableName string
	Registry  Registry
}

// New creates a new DynamoDB client.
func New(cfg aws.Config, tableName string, registry Registry) *DynamoDB {
	return &DynamoDB{
		Client:    dynamodb.New(cfg),
		TableName: tableName,
		Registry:  registry,
	}
}

// CreateTable creates the DynamoDB table.
func (d *DynamoDB) CreateTable(ctx context.Context, rcu, wcu int64) error {
	_, err := d.Client.CreateTableRequest(&dynamodb.CreateTableInput{
		TableName: aws.String(d.TableName),
		AttributeDefinitions: []dynamodb.AttributeDefinition{
			{AttributeName: aws.String("Project"), AttributeType: dynamodb.ScalarAttributeTypeS},
			{AttributeName: aws.String("ID"), AttributeType: dynamodb.ScalarAttributeTypeS},
		},
		KeySchema: []dynamodb.KeySchemaElement{
			{AttributeName: aws.String("Project"), KeyType: dynamodb.KeyTypeHash},
			{AttributeName: aws.String("ID"), KeyType: dynamodb.KeyTypeRange},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(rcu),
			WriteCapacityUnits: aws.Int64(wcu),
		},
	}).Send(ctx)
	if err != nil {
		return err
	}
	return nil
}

// PutResource creates or updates a resource.
func (d *DynamoDB) PutResource(ctx context.Context, project string, res *resource.Resource) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.TableName),
		Item: map[string]dynamodb.AttributeValue{
			"Project": attr.FromString(project),
			"ID":      attr.FromString(fmt.Sprintf("resource-%s", res.Name)),
			"Type":    attr.FromString(res.Type),
			"Name":    attr.FromString(res.Name),
			"Input":   attr.FromCtyValue(res.Input),
			"Output":  attr.FromCtyValue(res.Output),
		},
	}

	if len(res.Deps) > 0 {
		input.Item["Dependencies"] = attr.FromStringSet(res.Deps)
	}
	if len(res.Sources) > 0 {
		input.Item["Sources"] = attr.FromStringSet(res.Sources)
	}

	if _, err := d.Client.PutItemRequest(input).Send(ctx); err != nil {
		return errors.Wrap(err, "dynamodb put")
	}

	return nil
}

// DeleteResource deletes a resource. Returns an error if the resource does not exist.
func (d *DynamoDB) DeleteResource(ctx context.Context, project string, res *resource.Resource) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]dynamodb.AttributeValue{
			"Project": {S: aws.String(project)},
			"ID":      {S: aws.String(fmt.Sprintf("resource-%s", res.Name))},
		},
		ConditionExpression: aws.String("attribute_exists(ID)"),
	}
	_, err := d.Client.DeleteItemRequest(input).Send(ctx)
	if err != nil {
		return errors.Wrap(err, "dynamodb delete")
	}
	return nil
}

// ListResources lists all resources in a project. The order of the results is
// not guaranteed.
func (d *DynamoDB) ListResources(ctx context.Context, project string) ([]*resource.Resource, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.TableName),
		KeyConditionExpression: aws.String("#project = :project AND begins_with(#id, :prefix)"),
		ExpressionAttributeNames: map[string]string{
			"#project": "Project",
			"#id":      "ID",
		},
		ExpressionAttributeValues: map[string]dynamodb.AttributeValue{
			":project": {S: aws.String(project)},
			":prefix":  {S: aws.String("resource-")},
		},
	}
	resp, err := d.Client.QueryRequest(input).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "query dynamodb")
	}

	out := make([]*resource.Resource, *resp.Count)
	for i, item := range resp.QueryOutput.Items {
		res := &resource.Resource{}

		name, err := attr.ToString(item["Name"])
		if err != nil {
			return nil, fmt.Errorf("%d: field Name: %v", i, err)
		}
		res.Name = name

		typename, err := attr.ToString(item["Type"])
		if err != nil {
			return nil, fmt.Errorf("%d: field Type: %v", i, err)
		}
		res.Type = typename

		res.Deps = attr.ToStringSet(item["Dependencies"])
		res.Sources = attr.ToStringSet(item["Sources"])

		typ := d.Registry.Type(typename)
		if typ == nil {
			return nil, fmt.Errorf("%d: type %q not registered", i, typename)
		}
		fields := schema.Fields(typ)

		input, err := attr.ToCtyValue(item["Input"], fields.Inputs().CtyType())
		if err != nil {
			return nil, fmt.Errorf("%d: convert input: %v", i, err)
		}
		res.Input = input

		output, err := attr.ToCtyValue(item["Output"], fields.Outputs().CtyType())
		if err != nil {
			return nil, fmt.Errorf("%d: convert output: %v", i, err)
		}
		res.Output = output

		out[i] = res
	}

	return out, nil
}

// PutGraph creates or updates a graph.
func (d *DynamoDB) PutGraph(ctx context.Context, project string, g *graph.Graph) error {
	names := make([]string, 0, len(g.Resources))
	for name := range g.Resources {
		names = append(names, name)
	}
	sort.Strings(names)
	resources := make([]dynamodb.AttributeValue, len(names))
	for i, name := range names {
		res := g.Resources[name]
		item := map[string]dynamodb.AttributeValue{
			"Type":   attr.FromString(res.Type),
			"Name":   attr.FromString(res.Name),
			"Input":  attr.FromCtyValue(cty.UnknownAsNull(res.Input)),
			"Output": attr.FromCtyValue(cty.UnknownAsNull(res.Output)),
		}

		if len(res.Deps) > 0 {
			item["Dependencies"] = attr.FromStringSet(res.Deps)
		}
		if len(res.Sources) > 0 {
			item["Sources"] = attr.FromStringSet(res.Sources)
		}

		resources[i] = dynamodb.AttributeValue{M: item}
	}
	deps := make(map[string]dynamodb.AttributeValue, len(g.Dependencies))
	for name, dd := range g.Dependencies {
		depVals := make([]dynamodb.AttributeValue, len(dd))
		for i, d := range dd {
			dep := map[string]dynamodb.AttributeValue{
				"Field":      attr.FromCtyPath(d.Field),
				"Expression": attr.FromGraphExpression(d.Expression),
			}
			depVals[i] = dynamodb.AttributeValue{M: dep}
		}
		deps[name] = dynamodb.AttributeValue{L: depVals}
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.TableName),
		Item: map[string]dynamodb.AttributeValue{
			"Project": {S: aws.String(project)},
			"ID":      {S: aws.String("graph")},
		},
	}

	if len(resources) > 0 {
		input.Item["Resources"] = dynamodb.AttributeValue{L: resources}
	}
	if len(deps) > 0 {
		input.Item["Dependencies"] = dynamodb.AttributeValue{M: deps}
	}

	resp, err := d.Client.PutItemRequest(input).Send(ctx)
	if err != nil {
		return errors.Wrap(err, "dynamodb put")
	}
	_ = resp
	return nil
}

// GetGraph returns a graph for a project. Returns nil if the project does not
// have a graph.
func (d *DynamoDB) GetGraph(ctx context.Context, project string) (*graph.Graph, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]dynamodb.AttributeValue{
			"Project": {S: aws.String(project)},
			"ID":      {S: aws.String("graph")},
		},
	}
	resp, err := d.Client.GetItemRequest(input).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "dynamodb get")
	}
	if resp.Item == nil {
		// Not found
		return nil, nil
	}

	g := graph.New()

	for i, item := range resp.Item["Resources"].L {
		res := &resource.Resource{}

		name, err := attr.ToString(item.M["Name"])
		if err != nil {
			return nil, fmt.Errorf("%d: field Name: %v", i, err)
		}
		res.Name = name

		typename, err := attr.ToString(item.M["Type"])
		if err != nil {
			return nil, fmt.Errorf("%d: field Type: %v", i, err)
		}
		res.Type = typename

		res.Deps = attr.ToStringSet(item.M["Dependencies"])
		res.Sources = attr.ToStringSet(item.M["Sources"])

		typ := d.Registry.Type(typename)
		if typ == nil {
			return nil, fmt.Errorf("%d: type %q not registered", i, typename)
		}
		fields := schema.Fields(typ)

		input, err := attr.ToCtyValue(item.M["Input"], fields.Inputs().CtyType())
		if err != nil {
			return nil, fmt.Errorf("%d: convert input: %v", i, err)
		}
		res.Input = input

		output, err := attr.ToCtyValue(item.M["Output"], fields.Outputs().CtyType())
		if err != nil {
			return nil, fmt.Errorf("%d: convert output: %v", i, err)
		}
		res.Output = output

		g.AddResource(res)
	}
	for name, deps := range resp.Item["Dependencies"].M {
		for i, d := range deps.L {
			field, err := attr.ToCtyPath(d.M["Field"])
			if err != nil {
				return nil, fmt.Errorf("decode dependency field %s/%d: %v", name, i, err)
			}
			expr, err := attr.ToGraphExpression(d.M["Expression"])
			if err != nil {
				return nil, fmt.Errorf("decode dependency expression %s/%d: %v", name, i, err)
			}
			dep := graph.Dependency{
				Field:      field,
				Expression: expr,
			}
			g.AddDependency(name, dep)
		}
	}

	return g, nil
}
