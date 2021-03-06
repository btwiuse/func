package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
)

// APIGatewayResource provides a resource (GET /, POST /user etc) in a REST
// API.
type APIGatewayResource struct {
	// Inputs

	// The parent resource's identifier.
	ParentID string `func:"input"`

	// The last path segment for this resource.
	PathPart string `func:"input"`

	// The region the API Gateway is deployed to.
	Region string `func:"input"`

	// The string identifier of the associated RestApi.
	RestAPIID string `func:"input" name:"rest_api_id"`

	// Outputs

	// The resource's identifier.
	ID *string `func:"output"`

	// The full path for this resource.
	Path *string `func:"output"`

	apigatewayService
}

// Create creates a new resource.
func (p *APIGatewayResource) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.CreateResourceInput{
		ParentId:  aws.String(p.ParentID),
		PathPart:  aws.String(p.PathPart),
		RestApiId: aws.String(p.RestAPIID),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateResourceRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
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
		return err
	}

	input := &apigateway.DeleteResourceInput{
		ResourceId: p.ID,
		RestApiId:  aws.String(p.RestAPIID),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteResourceRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the rest api resource. Only the path part can be updated.
func (p *APIGatewayResource) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayResource)

	ops, err := apigatewaypatch.Resolve(
		prev, p,
		apigatewaypatch.Field{Name: "PathPart", Path: "/pathPart"},
	)
	if err != nil {
		return backoff.Permanent(err)
	}

	if len(ops) == 0 {
		return nil
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.UpdateResourceInput{
		RestApiId:       aws.String(p.RestAPIID),
		ResourceId:      prev.ID,
		PatchOperations: ops,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.UpdateResourceRequest(input).Send(ctx)
	return handlePutError(err)
}
