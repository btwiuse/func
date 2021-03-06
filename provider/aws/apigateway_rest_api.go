package aws

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// APIGatewayRestAPI provides a Serverless REST API.
type APIGatewayRestAPI struct {
	// Inputs

	// The source of the API key for metering requests according to a usage plan.
	//
	// Valid values are:
	// - HEADER to read the API key from the X-API-Key header of a request.
	// - AUTHORIZER to read the API key from the UsageIdentifierKey from a custom
	//   authorizer.
	APIKeySource *string `func:"input" validate:"oneof=HEADER AUTHORIZER"`

	// The list of binary media types supported by the RestApi.
	// By default, the RestApi supports only UTF-8-encoded text payloads.
	BinaryMediaTypes []string `func:"input"`

	// The ID of the RestApi that you want to clone from.
	CloneFrom *string `func:"input"`

	// The description of the RestApi.
	Description *string `func:"input"`

	// The endpoint configuration of this RestApi showing the endpoint types of
	// the API.
	EndpointConfiguration *struct {
		// A list of endpoint types of an API (RestApi) or its custom domain name (DomainName).
		//
		// - For an edge-optimized API and its custom domain name, the endpoint type is
		//   `EDGE`.
		// - For a regional API and its custom domain name, the endpoint type
		//   is `REGIONAL`. For a private API, the endpoint type is `PRIVATE`.
		Types []string
	} `func:"input"`

	// A nullable integer that is used to enable compression (with non-negative
	// between 0 and 10485760 (10M) bytes, inclusive) or disable compression (with
	// a null value) on an API. When compression is enabled, compression or decompression
	// is not applied on the payload if the payload size is smaller than this value.
	// Setting it to zero allows compression for any payload size.
	MinimumCompressionSize *int64 `func:"input" validate:"min=0,max=10485760"`

	// The name of the RestApi.
	Name string `func:"input"`

	// A stringified JSON policy document that applies to this RestApi regardless
	// of the caller and Method
	Policy *string `func:"input"`

	// The region the API Gateway is deployed to.
	Region string `func:"input"`

	// A version identifier for the API.
	Version *string `func:"input"`

	// Outputs

	// RFC3339 formatted date and time for when the API was created.
	CreatedDate string `func:"output"`

	// The API's identifier. This identifier is unique across all of your APIs in
	// API Gateway.
	ID *string `func:"output"`

	// The identifier for the API's root (/) resource.
	RootResourceID *string `func:"output"`

	apigatewayService
}

// Create creates a new rest api.
func (p *APIGatewayRestAPI) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.CreateRestApiInput{
		CloneFrom:              p.CloneFrom,
		Description:            p.Description,
		MinimumCompressionSize: p.MinimumCompressionSize,
		Name:                   aws.String(p.Name),
		Policy:                 p.Policy,
		Version:                p.Version,
		BinaryMediaTypes:       p.BinaryMediaTypes,
	}

	if p.APIKeySource != nil {
		input.ApiKeySource = apigateway.ApiKeySourceType(*p.APIKeySource)
	}
	if p.EndpointConfiguration != nil {
		types := make([]apigateway.EndpointType, len(p.EndpointConfiguration.Types))
		for i, t := range p.EndpointConfiguration.Types {
			types[i] = apigateway.EndpointType(t)
		}
		input.EndpointConfiguration = &apigateway.EndpointConfiguration{
			Types: types,
		}
	}

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateRestApiRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.CreatedDate = resp.CreatedDate.Format(time.RFC3339)
	p.ID = resp.Id

	// Read root resource
	rootReq := svc.GetResourcesRequest(&apigateway.GetResourcesInput{
		RestApiId: resp.Id,
	})
	rootRes, err := rootReq.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "read root resource")
	}
	for _, item := range rootRes.Items {
		if *item.Path == "/" {
			p.RootResourceID = item.Id
			return nil
		}
	}
	return errors.New("root resource not found")
}

// Delete removes a rest api.
func (p *APIGatewayRestAPI) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.DeleteRestApiInput{
		RestApiId: p.ID,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteRestApiRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the rest api.
//
// Fields that can be updated:
//
// - api_key_source
// - binary_media_type
// - description
// - endpoint_configuration.types
// - minimum_compression_size
// - name
func (p *APIGatewayRestAPI) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayRestAPI)

	// https://docs.aws.amazon.com/apigateway/api-reference/link-relation/restapi-update/
	ops, err := apigatewaypatch.Resolve(
		prev, p,
		apigatewaypatch.Field{Name: "APIKeySource", Path: "/apiKeySource"},
		apigatewaypatch.Field{Name: "BinaryMediaTypes", Path: "/binaryMediaTypes"},
		apigatewaypatch.Field{Name: "Description", Path: "/description"},
		apigatewaypatch.Field{Name: "MinimumCompressionSize", Path: "/minimumCompressionSize"},
		apigatewaypatch.Field{Name: "Name", Path: "/name"},
		apigatewaypatch.Field{Name: "Policy", Path: "/policy"},
		apigatewaypatch.Field{Name: "Version", Path: "/version"},
		apigatewaypatch.Field{
			Name: "EndpointConfiguration.Types",
			Path: "/endpointConfiguration/types",
			Modifier: func(ops []apigateway.PatchOperation) []apigateway.PatchOperation {
				p := *ops[0].Path
				slash := strings.LastIndex(p, "/")
				path := p[:slash] + "/0"

				// update or add will contain OpAdd
				for _, op := range ops {
					if op.Op == apigateway.OpAdd {
						value := p[slash+1:]

						// Set
						return []apigateway.PatchOperation{{
							Op:    apigateway.OpReplace,
							Path:  &path,
							Value: &value,
						}}
					}
				}

				// Reset to default (EDGE)
				edge := "EDGE"
				return []apigateway.PatchOperation{{
					Op:    apigateway.OpReplace,
					Path:  &path,
					Value: &edge,
				}}
			},
		},
	)
	if err != nil {
		return backoff.Permanent(errors.Wrap(err, "resolve patch"))
	}

	if len(ops) == 0 {
		return nil
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.UpdateRestApiInput{
		RestApiId:       prev.ID,
		PatchOperations: ops,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.UpdateRestApiRequest(input).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == apigateway.ErrCodeBadRequestException {
				if strings.Contains(aerr.Message(), "update is still in progress") {
					// Allow retry. Full error message:
					//   Unable to change the endpoint type while the previous
					//   endpoint type update is still in progress.
					return err
				}
			}
		}
		return handlePutError(err)
	}

	_ = resp

	return nil
}
