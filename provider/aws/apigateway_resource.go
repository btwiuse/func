package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// APIGatewayResource provides a resource (GET /, POST /user etc) in a REST
// API.
type APIGatewayResource struct {
	// Inputs

	// The parent resource's identifier.
	ParentID *string `func:"input,required"`

	// The last path segment for this resource.
	PathPart *string `func:"input,required"`

	// The region the API Gateway is deployed to.
	Region string `func:"input,required"`

	// The string identifier of the associated RestApi.
	RestAPIID *string `func:"input,required" name:"rest_api_id"`

	// Outputs

	// The resource's identifier.
	ID *string `func:"output"`

	// The full path for this resource.
	Path *string `func:"output"`

	apigatewayService
}

// Type returns the resource type of a apigateway resource.
func (p *APIGatewayResource) Type() string { return "aws_apigateway_resource" }

// Create creates a new resource.
func (p *APIGatewayResource) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.CreateResourceInput{
		ParentId:  p.ParentID,
		PathPart:  p.PathPart,
		RestApiId: p.RestAPIID,
	}

	req := svc.CreateResourceRequest(input)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}

	p.ID = resp.Id
	p.Path = resp.Path

	// The response contains ResourceMethods but they are never (can't be) set
	// when the resource is created. These values are only relevant when the
	// resource is read, but we have other ways of getting that information.
	// The ResourceMethods are omitted to keep the API surface of
	// APIGatewayResource smaller.

	return nil
}

// Delete removes a resource.
func (p *APIGatewayResource) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.DeleteResourceInput{
		ResourceId: p.ID,
		RestApiId:  p.RestAPIID,
	}

	req := svc.DeleteResourceRequest(input)
	if _, err := req.Send(ctx); err != nil {
		return err
	}

	return nil
}

// Update updates the rest api resource. Only the path part can be updated.
func (p *APIGatewayResource) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayResource)

	ops, err := apigatewaypatch.Resolve(
		prev, p,
		apigatewaypatch.Field{Name: "PathPart", Path: "/pathPart"},
	)
	if err != nil {
		return err
	}

	if len(ops) == 0 {
		return nil
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.UpdateResourceInput{
		RestApiId:       p.RestAPIID,
		ResourceId:      p.ID,
		PatchOperations: ops,
	}

	req := svc.UpdateResourceRequest(input)
	if _, err := req.Send(ctx); err != nil {
		return err
	}

	return nil
}
