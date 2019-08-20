package dynamodb

import (
	"context"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/schema"
	"github.com/pkg/errors"
	ctyjson "github.com/zclconf/go-cty/cty/json"
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
			{AttributeName: aws.String("Owner"), AttributeType: dynamodb.ScalarAttributeTypeS},
			{AttributeName: aws.String("ID"), AttributeType: dynamodb.ScalarAttributeTypeS},
		},
		KeySchema: []dynamodb.KeySchemaElement{
			{AttributeName: aws.String("Owner"), KeyType: dynamodb.KeyTypeHash},
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
func (d *DynamoDB) PutResource(ctx context.Context, ns, project string, resource resource.Resource) error {
	in, err := ctyjson.Marshal(resource.Input, resource.Input.Type())
	if err != nil {
		return errors.Wrap(err, "marshal input")
	}
	out, err := ctyjson.Marshal(resource.Output, resource.Output.Type())
	if err != nil {
		return errors.Wrap(err, "marshal output")
	}
	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.TableName),
		Item: map[string]dynamodb.AttributeValue{
			"Owner":  {S: aws.String(fmt.Sprintf("%s-%s", ns, project))},
			"ID":     {S: aws.String(fmt.Sprintf("resource-%s", resource.Name))},
			"Type":   {S: aws.String(resource.Type)},
			"Name":   {S: aws.String(resource.Name)},
			"Input":  {S: aws.String(string(in))},
			"Output": {S: aws.String(string(out))},
		},
	}
	if len(resource.Deps) > 0 {
		input.Item["Deps"] = dynamodb.AttributeValue{
			SS: resource.Deps,
		}
	}
	if len(resource.Sources) > 0 {
		input.Item["Sources"] = dynamodb.AttributeValue{
			SS: resource.Sources,
		}
	}
	resp, err := d.Client.PutItemRequest(input).Send(ctx)
	if err != nil {
		return errors.Wrap(err, "dynamodb put")
	}
	_ = resp
	return nil
}

// DeleteResource deletes a resource. No-op if the resource does not exist.
func (d *DynamoDB) DeleteResource(ctx context.Context, ns, project, name string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]dynamodb.AttributeValue{
			"Owner": {S: aws.String(fmt.Sprintf("%s-%s", ns, project))},
			"ID":    {S: aws.String(fmt.Sprintf("resource-%s", name))},
		},
	}
	_, err := d.Client.DeleteItemRequest(input).Send(ctx)
	if err != nil {
		return errors.Wrap(err, "dynamodb delete")
	}
	return nil
}

// ListResources lists all resources in a project.
func (d *DynamoDB) ListResources(ctx context.Context, ns, project string) (map[string]resource.Resource, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.TableName),
		KeyConditionExpression: aws.String("#owner = :owner AND begins_with(#id, :prefix)"),
		ExpressionAttributeNames: map[string]string{
			"#owner": "Owner",
			"#id":    "ID",
		},
		ExpressionAttributeValues: map[string]dynamodb.AttributeValue{
			":owner":  {S: aws.String(fmt.Sprintf("%s-%s", ns, project))},
			":prefix": {S: aws.String("resource-")},
		},
	}
	resp, err := d.Client.QueryRequest(input).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "query dynamodb")
	}
	out := make(map[string]resource.Resource, int(*resp.Count))
	for _, item := range resp.QueryOutput.Items {
		typename := *item["Type"].S
		typ := d.Registry.Type(typename)
		if typ == nil {
			return nil, fmt.Errorf("type %q not registered", typename)
		}
		fields := schema.Fields(typ)

		input, err := ctyjson.Unmarshal([]byte(*item["Input"].S), fields.Inputs().CtyType())
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal input")
		}

		output, err := ctyjson.Unmarshal([]byte(*item["Output"].S), fields.Outputs().CtyType())
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal output")
		}

		res := resource.Resource{
			Name:    *item["Name"].S,
			Type:    typename,
			Input:   input,
			Output:  output,
			Deps:    item["Deps"].SS,
			Sources: item["Sources"].SS,
		}

		out[res.Name] = res
	}
	return out, nil
}

// PutGraph creates or updates a graph.
func (d *DynamoDB) PutGraph(ctx context.Context, ns, project string, graph *graph.Graph) error {
	data, err := graph.MarshalJSON()
	if err != nil {
		return errors.Wrap(err, "marshal graph")
	}
	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.TableName),
		Item: map[string]dynamodb.AttributeValue{
			"Owner": {S: aws.String(fmt.Sprintf("%s-%s", ns, project))},
			"ID":    {S: aws.String("graph")},
			"Data":  {S: aws.String(string(data))},
		},
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
func (d *DynamoDB) GetGraph(ctx context.Context, ns, project string) (*graph.Graph, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]dynamodb.AttributeValue{
			"Owner": {S: aws.String(fmt.Sprintf("%s-%s", ns, project))},
			"ID":    {S: aws.String("graph")},
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
	dec := graph.JSONDecoder{
		Target:   graph.New(),
		Registry: d.Registry,
	}
	data := *resp.Item["Data"].S
	if err := dec.UnmarshalJSON([]byte(data)); err != nil {
		return nil, errors.Wrap(err, "unmarshal graph")
	}
	return dec.Target, nil
}
