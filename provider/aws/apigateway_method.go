//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct APIGatewayMethod --template ../../tools/structdoc/template.txt --data type=aws_apigateway_method --output ../../docs/resources/aws/apigateway_method.md

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// APIGatewayMethod provides a resource (`GET /`, `POST /user` etc) in a REST
// API.
type APIGatewayMethod struct {
	// Inputs

	// Specifies whether the method requires a valid API key.
	APIKeyRequired *bool `input:"api_key_required"`

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
	AuthorizationScopes *[]string `input:"authorization_scopes"`

	// The method's authorization type.
	//
	// Valid values:
	// - `NONE`: open access
	// - `AWS_IAM`: use IAM permissions
	// - `CUSTOM`: use a custom authorizer
	// - `COGNITO_USER_POOL`: use a [Cognito](https://aws.amazon.com/cognito/) user pool
	AuthorizationType string `input:"authorization_type"`

	// Specifies the identifier of an Authorizer to use on this Method, if the type
	// is `CUSTOM` or `COGNITO_USER_POOL`. The authorizer identifier is generated by
	// API Gateway when you created the authorizer.
	AuthorizerID *string `input:"authorizer_id"`

	// Specifies the method request's HTTP method type.
	HTTPMethod string `input:"http_method"`

	// A human-friendly operation identifier for the method.
	//
	// For example, you can assign the `operation_name` of `ListPets` for the `GET /pets`
	// method in [PetStore](https://petstore-demo-endpoint.execute-api.com/petstore/pets) example.
	OperationName *string `input:"operation_name"`

	// The region the API Gateway is deployed to.
	Region string `input:"region"`

	// Specifies the Model resources used for the request's content type. Request
	// models are represented as a key/value map, with a content type as the key
	// and a Model name as the value.
	RequestModels *map[string]string `input:"request_models"`

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
	RequestParameters *map[string]bool `input:"request_parameters"`

	// The identifier of a RequestValidator for validating the method request.
	RequestValidatorID *string `input:"request_validator_id"`

	// The Resource identifier for the new Method resource.
	ResourceID string `input:"resource_id"`

	// The string identifier of the associated RestApi.
	RestAPIID string `input:"rest_api_id"`

	// No outputs

	apigatewayService
}

// Type returns the resource type of a apigateway resource.
func (p *APIGatewayMethod) Type() string { return "aws_apigateway_method" }

// Create creates a new resource.
func (p *APIGatewayMethod) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.PutMethodInput{
		ApiKeyRequired:     p.APIKeyRequired,
		AuthorizationType:  aws.String(p.AuthorizationType),
		AuthorizerId:       p.AuthorizerID,
		HttpMethod:         aws.String(p.HTTPMethod),
		OperationName:      p.OperationName,
		RequestValidatorId: p.RequestValidatorID,
		ResourceId:         aws.String(p.ResourceID),
		RestApiId:          aws.String(p.RestAPIID),
	}

	if p.AuthorizationScopes != nil {
		input.AuthorizationScopes = *p.AuthorizationScopes
	}
	if p.RequestModels != nil {
		input.RequestModels = *p.RequestModels
	}
	if p.RequestParameters != nil {
		input.RequestParameters = *p.RequestParameters
	}

	req := svc.PutMethodRequest(input)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
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
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteMethodRequest(&apigateway.DeleteMethodInput{
		HttpMethod: aws.String(p.HTTPMethod),
		ResourceId: aws.String(p.ResourceID),
		RestApiId:  aws.String(p.RestAPIID),
	})
	if _, err := req.Send(ctx); err != nil {
		return err
	}

	return nil
}

// Update updates the rest api resource. Only the path part can be updated.
func (p *APIGatewayMethod) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayMethod)

	if prev.HTTPMethod != p.HTTPMethod ||
		prev.ResourceID != p.ResourceID ||
		prev.RestAPIID != p.RestAPIID {
		// These cannot be updated with patch.
		if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
			return errors.Wrap(err, "update-delete")
		}
		if err := p.Create(ctx, r.CreateRequest()); err != nil {
			return errors.Wrap(err, "update-create")
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
		return err
	}

	if len(ops) == 0 {
		return nil
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.UpdateMethodRequest(&apigateway.UpdateMethodInput{
		HttpMethod:      aws.String(p.HTTPMethod),
		ResourceId:      aws.String(p.ResourceID),
		RestApiId:       aws.String(p.RestAPIID),
		PatchOperations: ops,
	})
	if _, err := req.Send(ctx); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "SerializationError" {
				return backoff.Permanent(err)
			}
		}
		return err
	}

	return nil
}
