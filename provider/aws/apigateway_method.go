package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
)

// APIGatewayMethod provides a resource (`GET /`, `POST /user` etc) in a REST
// API.
type APIGatewayMethod struct {
	// Inputs

	// Specifies whether the method requires a valid API key.
	APIKeyRequired *bool `func:"input"`

	// A list of authorization scopes configured on the method. The scopes are used
	// with a `COGNITO_USER_POOLS` authorizer to authorize the method invocation.
	//
	// The authorization works by matching the method scopes against the scopes
	// parsed from the access token in the incoming request. The method invocation
	// is authorized if any method scopes matches a claimed scope in the access
	// token. Otherwise, the invocation is not authorized.
	//
	// When the method scope is configured, the client must provide an access
	// token instead of an identity token for authorization purposes.
	AuthorizationScopes []string `func:"input"`

	// The method's authorization type.
	//
	// Valid values:
	// - `NONE`: open access
	// - `AWS_IAM`: use IAM permissions
	// - `CUSTOM`: use a custom authorizer
	// - `COGNITO_USER_POOL`: use a [Cognito](https://aws.amazon.com/cognito/) user pool
	AuthorizationType string `func:"input" validate:"oneof=NONE AWS_IAM CUSTOM COGNITO_USER_POOL"`

	// Specifies the identifier of an Authorizer to use on this Method, if the type
	// is `CUSTOM` or `COGNITO_USER_POOL`. The authorizer identifier is generated by
	// API Gateway when you created the authorizer.
	AuthorizerID *string `func:"input"`

	// Specifies the method request's HTTP method type.
	HTTPMethod string `func:"input"`

	// A human-friendly operation identifier for the method.
	//
	// For example, you can assign the `operation_name` of `ListPets` for the `GET /pets`
	// method in [PetStore](https://petstore-demo-endpoint.execute-api.com/petstore/pets) example.
	OperationName *string `func:"input"`

	// The region the API Gateway is deployed to.
	Region string `func:"input"`

	// Specifies the Model resources used for the request's content type. Request
	// models are represented as a key/value map, with a content type as the key
	// and a Model name as the value.
	RequestModels map[string]string `func:"input"`

	// A key-value map defining required or optional method request parameters that
	// can be accepted by API Gateway.
	//
	// A key defines a method request parameter name matching the pattern of
	// `method.request.{location}.{name}`, where location is `querystring`,
	// `path`, or `header` and name is a valid and unique parameter name. The
	// value associated with the key is a Boolean flag indicating whether the
	// parameter is required (`true`) or optional (`false`). The method request
	// parameter names defined here are available in Integration to be mapped
	// to integration request parameters or body-mapping templates.
	RequestParameters map[string]bool `func:"input"`

	// The identifier of a RequestValidator for validating the method request.
	RequestValidatorID *string `func:"input"`

	// The Resource identifier for the new Method resource.
	ResourceID string `func:"input"`

	// The string identifier of the associated Rest API.
	RestAPIID string `func:"input" name:"rest_api_id"`

	// No outputs

	apigatewayService
}

// Create creates a new resource.
func (p *APIGatewayMethod) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.PutMethodInput{
		ApiKeyRequired:      p.APIKeyRequired,
		AuthorizationType:   aws.String(p.AuthorizationType),
		AuthorizerId:        p.AuthorizerID,
		HttpMethod:          aws.String(p.HTTPMethod),
		OperationName:       p.OperationName,
		RequestValidatorId:  p.RequestValidatorID,
		ResourceId:          aws.String(p.ResourceID),
		RestApiId:           aws.String(p.RestAPIID),
		AuthorizationScopes: p.AuthorizationScopes,
		RequestModels:       p.RequestModels,
		RequestParameters:   p.RequestParameters,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.PutMethodRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	// The response is a UpdateMethodOutput but it does not contain any
	// additional information from create.
	_ = resp

	return nil
}

// Delete removes a resource.
func (p *APIGatewayMethod) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.DeleteMethodInput{
		HttpMethod: aws.String(p.HTTPMethod),
		ResourceId: aws.String(p.ResourceID),
		RestApiId:  aws.String(p.RestAPIID),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteMethodRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the rest api resource. Only the path part can be updated.
func (p *APIGatewayMethod) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayMethod)

	if prev.HTTPMethod != p.HTTPMethod ||
		prev.ResourceID != p.ResourceID ||
		prev.RestAPIID != p.RestAPIID {
		// These cannot be updated with patch.
		if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
			return err
		}
		if err := p.Create(ctx, r.CreateRequest()); err != nil {
			return err
		}

		// No further patch operations are needed, since the newly created
		// resource captured all changes.
		return nil
	}

	ops, err := apigatewaypatch.Resolve(
		prev, p,
		apigatewaypatch.Field{Name: "APIKeyRequired", Path: "/apiKeyRequired"},
		apigatewaypatch.Field{Name: "AuthorizationScopes", Path: "/authorizationScopes"},
		apigatewaypatch.Field{Name: "AuthorizationType", Path: "/authorizationType"},
		apigatewaypatch.Field{Name: "AuthorizerID", Path: "/authorizerId"},
		apigatewaypatch.Field{Name: "OperationName", Path: "/operationName"},
		apigatewaypatch.Field{Name: "RequestModels", Path: "/requestModels"},
		apigatewaypatch.Field{Name: "RequestParameters", Path: "/requestParameters"},
		apigatewaypatch.Field{Name: "RequestValidatorID", Path: "/requestValidatorId"},
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

	input := &apigateway.UpdateMethodInput{
		HttpMethod:      aws.String(p.HTTPMethod),
		ResourceId:      aws.String(p.ResourceID),
		RestApiId:       aws.String(p.RestAPIID),
		PatchOperations: ops,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.UpdateMethodRequest(input).Send(ctx)
	return handlePutError(err)
}
