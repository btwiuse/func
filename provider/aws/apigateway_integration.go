package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
)

// APIGatewayIntegration provides a resource (GET /, POST /user etc) in a REST
// API.
type APIGatewayIntegration struct {
	// Specifies a put integration input's cache key parameters.
	CacheKeyParameters []string `func:"input"`

	// Specifies a put integration input's cache namespace.
	CacheNamespace *string `func:"input"`

	// The [id](https://docs.aws.amazon.com/apigateway/api-reference/resource/vpc-link/#id)
	// of the VpcLink used for the integration when `connectionType=VPC_LINK` and
	// undefined, otherwise.
	ConnectionID *string `func:"input"`

	// The type of the network connection to the integration endpoint. The valid
	// value is `INTERNET` for connections through the public routable internet or
	// `VPC_LINK` for private connections between API Gateway and a network load balancer
	// in a VPC. The default value is `INTERNET`.
	ConnectionType *string `func:"input" validate:"oneof=INTERNET VPC_LINK"`

	// Specifies how to handle request payload content type conversions. Supported
	// values are `CONVERT_TO_BINARY` and `CONVERT_TO_TEXT`, with the following
	// behaviors:
	//
	// - `CONVERT_TO_BINARY`: Converts a request payload from a Base64-encoded
	//   string to the corresponding binary blob.
	// - `CONVERT_TO_TEXT`: Converts a request payload from a binary blob to a
	//   Base64-encoded string.
	//
	// If this property is not defined, the request payload will be passed through
	// from the method request to integration request without modification, provided
	// that the `passthrough_behaviors` is configured to support payload pass-through.
	ContentHandling *string `func:"input" validate:"oneof=CONVERT_TO_BINARY CONVERT_TO_TEXT"`

	// Specifies the credentials required for the integration, if any.
	//
	// For AWS integrations, three options are available:
	// - To specify an IAM Role for API Gateway to assume, use the role's Amazon Resource Name (ARN)
	// - To require that the caller's identity be passed through from the
	//   request, specify the string `arn:aws:iam::*:user/*`
	// - To use resource-based permissions on supported AWS services, leave this field blank.
	Credentials *string `func:"input"`

	// Specifies a put integration request's HTTP method.
	HTTPMethod string `func:"input"`

	// Specifies a put integration HTTP method.
	//
	// When the integration type is `HTTP` or `AWS`, this field is required.
	IntegrationHTTPMethod *string `func:"input"`

	// Specifies the pass-through behavior for incoming requests based on the
	// Content-Type header in the request, and the available mapping templates
	// specified as the `request_templates` property on the Integration
	// resource.
	//
	// There are three valid values:
	//
	// - `WHEN_NO_MATCH` passes the request body for unmapped content types through
	//   to the integration back end without transformation.
	// - `NEVER` rejects unmapped content types with an HTTP 415 'Unsupported Media
	//   Type' response.
	// - `WHEN_NO_TEMPLATES` allows pass-through when the integration has NO content
	//   types mapped to templates. However if there is at least one content type
	//   defined, unmapped content types will be rejected with the same 415 response.
	PassthroughBehavior *string `func:"input" validate:"oneof=WHEN_NO_MATCH NEVER WHEN_NO_TEMPLATES"`

	// The region the API Gateway is deployed to.
	Region string `func:"input"`

	// A key-value map specifying request parameters that are passed from the
	// method request to the back end. The key is an integration request
	// parameter name and the associated value is a method request parameter
	// value or static value that must be enclosed within single quotes and
	// pre-encoded as required by the back end. The method request parameter
	// value must match the pattern of `method.request.{location}.{name}`,
	// where location is `querystring`, `path`, or `header` and name must be a
	// valid and unique method request parameter name.
	RequestParameters map[string]string `func:"input"`

	// Represents a map of
	// [Velocity](https://velocity.apache.org/engine/1.7/user-guide.html#velocity-template-language-vtl-an-introduction)
	// templates that are applied on the request payload based on the value of
	// the Content-Type header sent by the client. The content type value is
	// the key in this map, and the template (as a String) is the value.
	RequestTemplates map[string]string `func:"input"`

	// Specifies a put integration request's resource ID.
	ResourceID string `func:"input"`

	// The string identifier of the associated Rest API.
	RestAPIID string `func:"input" name:"rest_api_id"`

	// Custom timeout between 50 and 29,000 milliseconds. The default value is 29,000
	// milliseconds or 29 seconds.
	TimeoutInMillis *int64 `func:"input" validate:"gte=50,lte=29000"`

	// Specifies a put integration input's type.
	//
	// Valid values:
	// - `AWS`: for integrating the API method request with an AWS service
	//   action, including the Lambda function-invoking action. With the Lambda
	//   function-invoking action, this is referred to as the Lambda custom
	//   integration. With any other AWS service action, this is known as AWS
	//   integration.
	// - `AWS_PROXY`: for integrating the API method request with the Lambda
	//   function-invoking action with the client request passed through as-is.
	//   This integration is also referred to as the Lambda proxy integration.
	// - `HTTP`: for integrating the API method request with an HTTP endpoint,
	//   including a private HTTP endpoint within a VPC. This integration is also
	//   referred to as the HTTP custom integration.
	// - `HTTP_PROXY`: for integrating the API method request with an HTTP
	//   endpoint, including a private HTTP endpoint within a VPC, with the
	//   client request passed through as-is. This is also referred to as the
	//   `HTTP` proxy integration.
	// - `MOCK`: for integrating the API method request with API Gateway as a
	//   "loop-back" endpoint without invoking any backend.
	//
	// For the HTTP and HTTP proxy integrations, each integration can specify a
	// protocol (http/https), port and path. Standard 80 and 443 ports are
	// supported as well as custom ports above 1024. An HTTP or HTTP proxy
	// integration with a `connection_type` of `VPC_LINK` is referred to as a
	// private integration and uses a VpcLink to connect API Gateway to a
	// network load balancer of a VPC.
	IntegrationType string `func:"input" validate:"oneof=AWS AWS_PROXY HTTP HTTP_PROXY MOCK"`

	// Specifies Uniform Resource Identifier (URI) of the integration endpoint.
	//
	// - For `HTTP` or `HTTP_PROXY` integrations, the URI must be a fully
	//   formed, encoded HTTP(S) URL according to the RFC-3986 specification, for
	//   either standard integration, where connectionType is not `VPC_LINK`, or
	//   private integration, where connectionType is `VPC_LINK`. For a private
	//   HTTP integration, the URI is not used for routing.
	// - For AWS or AWS_PROXY integrations, the URI is of the form
	//   `arn:aws:apigateway:{region}:{subdomain.service|service}:path|action/{service_api}`.
	//   Here, `{Region}` is the API Gateway region (e.g., _us-east-1_);
	//   `{service}` is the name of the integrated AWS service (e.g., _s3_);
	//   and `{subdomain}` is a designated subdomain supported by certain AWS
	//   service for fast host-name lookup. The `action` can be used for an AWS
	//   service action-based API, using an `Action={name}&{p1}={v1}&p2={v2}...`
	//   query string. The ensuing `{service_api}` refers to a supported action
	//   `{name}` plus any required input parameters. Alternatively, `path` can
	//   be used for an AWS service path-based API. The ensuing `service_api`
	//   refers to the path to an AWS service resource, including the region of
	//   the integrated AWS service, if applicable.  For example, for
	//   integration with the S3 API of
	//   [GetObject](https://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectGET.html),
	//   the uri can be either
	//   `arn:aws:apigateway:us-west-2:s3:action/GetObject&Bucket={bucket}&Key={key}`
	//   or `arn:aws:apigateway:us-west-2:s3:path/{bucket}/{key}`
	URI *string `func:"input"`

	// Outputs

	// Specifies the integration's responses.
	//
	// The key in the map is the HTTP status code.
	IntegrationResponses map[string]APIGatewayIntegrationResponse `func:"output"`

	apigatewayService
}

// APIGatewayIntegrationResponse is the output from an integration for a
// certain HTTP status code. It is provided as an output from
// IntegrationResponses.
type APIGatewayIntegrationResponse struct {
	// Specifies how to handle response payload content type conversions.
	// Supported values are CONVERT_TO_BINARY and CONVERT_TO_TEXT, with the
	// following behaviors:
	//
	// - `CONVERT_TO_BINARY`: Converts a response payload from a Base64-encoded
	//    string to the corresponding binary blob.
	// - `CONVERT_TO_TEXT`: Converts a response payload from a binary blob to a
	//    Base64-encoded string.
	//
	// If this property is not defined, the response payload will be passed
	// through from the integration response to the method response without
	// modification.
	ContentHandling string

	// A key-value map specifying response parameters that are passed to the
	// method response from the back end. The key is a method response header
	// parameter name and the mapped value is an integration response header
	// value, a static value enclosed within a pair of single quotes, or a JSON
	// expression from the integration response body. The mapping key must
	// match the pattern of `method.response.header.{name}`, where name is a
	// valid and unique header name.  The mapped non-static value must match
	// the pattern of `integration.response.header.{name}` or
	// `integration.response.body.{JSON-expression}`, where name is a valid and
	// unique response header name and JSON-expression is a valid JSON
	// expression without the `$` prefix.
	ResponseParameters map[string]string

	// Specifies the templates used to transform the integration response body.
	// Response templates are represented as a key/value map, with a
	// content-type as the key and a template as the value.
	ResponseTemplates map[string]string

	// Specifies the regular expression pattern used to choose an integration
	// response based on the response from the back end.
	//
	// For example, if the success response returns nothing and the error
	// response returns some string, you could use the `.+` regex to match
	// error response. However, make sure that the error response does not
	// contain any newline (`\n`) character in such cases. If the back end is
	// an AWS Lambda function, the AWS Lambda function error header is matched.
	// For all other HTTP and AWS back ends, the HTTP status code is matched.
	SelectionPattern string

	// Specifies the status code that is used to map the integration response
	// to an existing MethodResponse.
	StatusCode string
}

// Create creates a new resource.
func (p *APIGatewayIntegration) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.PutIntegrationInput{
		CacheNamespace:        p.CacheNamespace,
		CacheKeyParameters:    p.CacheKeyParameters,
		ConnectionId:          p.ConnectionID,
		Credentials:           p.Credentials,
		HttpMethod:            aws.String(p.HTTPMethod),
		IntegrationHttpMethod: p.IntegrationHTTPMethod,
		PassthroughBehavior:   p.PassthroughBehavior,
		ResourceId:            aws.String(p.ResourceID),
		RequestParameters:     p.RequestParameters,
		RequestTemplates:      p.RequestTemplates,
		RestApiId:             aws.String(p.RestAPIID),
		TimeoutInMillis:       p.TimeoutInMillis,
		Type:                  apigateway.IntegrationType(p.IntegrationType),
		Uri:                   p.URI,
	}

	if p.ConnectionType != nil {
		input.ConnectionType = apigateway.ConnectionType(*p.ConnectionType)
	}
	if p.ContentHandling != nil {
		input.ContentHandling = apigateway.ContentHandlingStrategy(*p.ContentHandling)
	}

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.PutIntegrationRequest(input).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok {
			if aerr.Code() == apigateway.ErrCodeNotFoundException {
				// Retry
				return err
			}
		}
		return handlePutError(err)
	}

	p.IntegrationResponses = make(map[string]APIGatewayIntegrationResponse, len(resp.IntegrationResponses))
	for k, ir := range resp.IntegrationResponses {
		p.IntegrationResponses[k] = APIGatewayIntegrationResponse{
			ContentHandling:    string(ir.ContentHandling),
			ResponseParameters: ir.ResponseParameters,
			ResponseTemplates:  ir.ResponseTemplates,
			SelectionPattern:   *ir.SelectionPattern,
			StatusCode:         *ir.StatusCode,
		}
	}

	return nil
}

// Delete removes a resource.
func (p *APIGatewayIntegration) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.DeleteIntegrationInput{
		HttpMethod: aws.String(p.HTTPMethod),
		ResourceId: aws.String(p.ResourceID),
		RestApiId:  aws.String(p.RestAPIID),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteIntegrationRequest(input).Send(ctx)
	if err != nil {
		return handleDelError(err)
	}

	return nil
}

// Update updates the rest api resource. Only the path part can be updated.
func (p *APIGatewayIntegration) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayIntegration)

	if prev.ResourceID != p.ResourceID ||
		prev.RestAPIID != p.RestAPIID ||
		prev.HTTPMethod != p.HTTPMethod ||
		prev.IntegrationType != p.IntegrationType {
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
		apigatewaypatch.Field{Name: "CacheKeyParameters", Path: "/cacheKeyParameters"},
		apigatewaypatch.Field{Name: "CacheNamespace", Path: "/cacheNamespace"},
		apigatewaypatch.Field{Name: "ConnectionID", Path: "/connectionId"},
		apigatewaypatch.Field{Name: "ConnectionType", Path: "/connectiontype"},
		apigatewaypatch.Field{Name: "ContentHandling", Path: "/contentHandling"},
		apigatewaypatch.Field{Name: "Credentials", Path: "/credentials"},
		apigatewaypatch.Field{Name: "IntegrationHTTPMethod", Path: "/http_method"},
		apigatewaypatch.Field{Name: "PassthroughBehavior", Path: "/passthroughBehavior"},
		apigatewaypatch.Field{Name: "RequestParameters", Path: "/requestParameters"},
		apigatewaypatch.Field{Name: "Timeout", Path: "/timeoutInMillis"},
		apigatewaypatch.Field{Name: "URI", Path: "/uri"},
		apigatewaypatch.Field{Name: "Timeout", Path: "/timeoutInMillis"},
	)
	if err != nil {
		return err
	}

	if len(ops) == 0 {
		return nil
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &apigateway.UpdateIntegrationInput{
		HttpMethod:      aws.String(p.HTTPMethod),
		ResourceId:      aws.String(p.ResourceID),
		RestApiId:       aws.String(p.RestAPIID),
		PatchOperations: ops,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.UpdateIntegrationRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.IntegrationResponses = make(map[string]APIGatewayIntegrationResponse, len(resp.IntegrationResponses))
	for k, ir := range resp.IntegrationResponses {
		p.IntegrationResponses[k] = APIGatewayIntegrationResponse{
			ContentHandling:    string(ir.ContentHandling),
			ResponseParameters: ir.ResponseParameters,
			ResponseTemplates:  ir.ResponseTemplates,
			SelectionPattern:   *ir.SelectionPattern,
			StatusCode:         *ir.StatusCode,
		}
	}

	return nil
}
