package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/apigatewaypatch"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// APIGatewayStage provides a stage for an APIGateway Deployment.
type APIGatewayStage struct {
	// Inputs

	// Whether cache clustering is enabled for the stage.
	CacheClusterEnabled *bool `func:"input"`

	// Specifies the cache cluster size for the Stage resource specified in the
	// input, if a cache cluster is enabled.
	CacheClusterSize *string `func:"input"`

	// The input configuration for the canary deployment when the deployment is
	// a canary release deployment.
	CanarySettings *APIGatewayCanarySettings `func:"input"`

	// The identifier of the Deployment resource for the Stage resource.
	DeploymentID string `func:"input"`

	// The description of the Stage resource.
	Description *string `func:"input"`

	// The version of the associated API documentation.
	DocumentationVersion *string `func:"input"`

	// The region the API Gateway is deployed to.
	Region string `func:"input"`

	// The string identifier of the associated Rest API.
	RestAPIID string `func:"input," name:"rest_api_id"`

	// The name for the Stage resource.
	StageName string `func:"input"`

	// The key-value map of strings. The valid character set is `[a-zA-Z+-=._:/]`.
	// The tag key can be up to 128 characters and must not start with aws:. The
	// tag value can be up to 256 characters.
	Tags map[string]string `func:"input"`

	// Specifies whether active tracing with X-ray is enabled for the Stage.
	TracingEnabled *bool `func:"input"`

	// A map that defines the stage variables for the new Stage resource. Variable
	// names can have alphanumeric and underscore characters, and the values must
	// match `[A-Za-z0-9-._~:/?#&=,]+`.
	Variables map[string]string `func:"input"`

	// Outputs

	// Settings for logging access in this stage.
	AccessLogSettings *AccessLogSettings `func:"output"`

	// The status of the cache cluster for the stage, if enabled.
	//
	// Value will be one of:
	//   - `CREATE_IN_PROGRESS`
	//   - `AVAILABLE`
	//   - `DELETE_IN_PROGRESS`
	//   - `NOT_AVAILABLE`
	//   - `FLUSH_IN_PROGRESS`
	CacheClusterStatus string `func:"output"`

	// The identifier of a client certificate for an API stage.
	ClientCertificateID string `func:"output"`

	// The timestamp when the stage was created.
	CreatedDate time.Time `func:"output"`

	// The timestamp when the stage last updated.
	LastUpdatedDate time.Time `func:"output"`

	// A map that defines the method settings for a Stage resource. Keys (designated
	// as /{method_setting_key below) are method paths defined as {resource_path}/{http_method}
	// for an individual method override, or /\*/\* for overriding all methods in
	// the stage.
	MethodSettings map[string]MethodSetting `func:"output"`

	// The ARN of the WebAcl associated with the Stage.
	WebACLARN *string `func:"output" name:"web_acl_arn"`

	apigatewayService
}

// APIGatewayCanarySettings contains settings for canary deployment.
type APIGatewayCanarySettings struct {
	// The ID of the canary deployment.
	DeploymentID *string

	// The percent (0-100) of traffic diverted to a canary deployment.
	PercentTraffic *float64

	// Stage variables overridden for a canary release deployment, including new
	// stage variables introduced in the canary. These stage variables are represented
	// as a string-to-string map between stage variable names and their values.
	StageVariableOverrides *map[string]string

	// A Boolean flag to indicate whether the canary deployment uses the stage cache
	// or not.
	UseStageCache *bool
}

// AccessLogSettings contains settings for an APIGateway deployment stage.
type AccessLogSettings struct {
	// The ARN of the CloudWatch Logs log group to receive access logs.
	DestinationARN *string

	// A single line format of the access logs of data, as specified by
	// selected `$context` variables.
	// The format must include at least `$context.requestId`.
	Format *string
}

// MethodSetting contains settings for a method in an APIGateway deployment stage.
type MethodSetting struct { // nolint: maligned
	// Specifies whether the cached responses are encrypted. The PATCH path for
	// this setting is /{method_setting_key}/caching/dataEncrypted, and the value
	// is a Boolean.
	CacheDataEncrypted bool

	// Specifies the time to live (TTL), in seconds, for cached responses. The higher
	// the TTL, the longer the response will be cached. The PATCH path for this
	// setting is /{method_setting_key}/caching/ttlInSeconds, and the value is an
	// integer.
	CacheTTLInSeconds int64

	// Specifies whether responses should be cached and returned for requests. A
	// cache cluster must be enabled on the stage for responses to be cached. The
	// PATCH path for this setting is /{method_setting_key}/caching/enabled, and
	// the value is a Boolean.
	CachingEnabled bool

	// Specifies whether data trace logging is enabled for this method, which affects
	// the log entries pushed to Amazon CloudWatch Logs. The PATCH path for this
	// setting is /{method_setting_key}/logging/dataTrace, and the value is a Boolean.
	DataTraceEnabled bool

	// Specifies the logging level for this method, which affects the log entries
	// pushed to Amazon CloudWatch Logs. The PATCH path for this setting is /{method_setting_key}/logging/loglevel,
	// and the available levels are OFF, ERROR, and INFO.
	LoggingLevel string

	// Specifies whether Amazon CloudWatch metrics are enabled for this method.
	// The PATCH path for this setting is /{method_setting_key}/metrics/enabled,
	// and the value is a Boolean.
	MetricsEnabled bool

	// Specifies whether authorization is required for a cache invalidation request.
	// The PATCH path for this setting is /{method_setting_key}/caching/requireAuthorizationForCacheControl,
	// and the value is a Boolean.
	RequireAuthorizationForCacheControl bool

	// Specifies the throttling burst limit. The PATCH path for this setting is
	// /{method_setting_key}/throttling/burstLimit, and the value is an integer.
	ThrottlingBurstLimit int64

	// Specifies the throttling rate limit. The PATCH path for this setting is /{method_setting_key}/throttling/rateLimit,
	// and the value is a double.
	ThrottlingRateLimit float64

	// Specifies how to handle unauthorized requests for cache invalidation.
	// Available values are:
	// - `FAIL_WITH_403`
	// - `SUCCEED_WITH_RESPONSE_HEADER`
	// - `SUCCEED_WITHOUT_RESPONSE_HEADER`
	UnauthorizedCacheControlHeaderStrategy string
}

// Type returns the resource type of an apigateway deployment.
func (p *APIGatewayStage) Type() string { return "aws_apigateway_stage" }

// Create creates a new deployment.
func (p *APIGatewayStage) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.CreateStageInput{
		DeploymentId:         aws.String(p.DeploymentID),
		Description:          p.Description,
		DocumentationVersion: p.DocumentationVersion,
		RestApiId:            aws.String(p.RestAPIID),
		StageName:            aws.String(p.StageName),
		TracingEnabled:       p.TracingEnabled,
		Tags:                 p.Tags,
		Variables:            p.Variables,
	}

	if p.CacheClusterSize != nil {
		input.CacheClusterSize = apigateway.CacheClusterSize(*p.CacheClusterSize)
	}
	if p.CanarySettings != nil {
		input.CanarySettings = &apigateway.CanarySettings{
			DeploymentId:   p.CanarySettings.DeploymentID,
			PercentTraffic: p.CanarySettings.PercentTraffic,
			UseStageCache:  p.CanarySettings.UseStageCache,
		}
		if p.CanarySettings.StageVariableOverrides != nil {
			input.CanarySettings.StageVariableOverrides = make(map[string]string)
			for k, v := range *p.CanarySettings.StageVariableOverrides {
				input.CanarySettings.StageVariableOverrides[k] = v
			}
		}
	}

	req := svc.CreateStageRequest(input)
	resp, err := req.Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == apigateway.ErrCodeBadRequestException {
				return backoff.Permanent(err)
			}
		}
		return err
	}

	if resp.AccessLogSettings != nil {
		p.AccessLogSettings = &AccessLogSettings{
			DestinationARN: resp.AccessLogSettings.DestinationArn,
			Format:         resp.AccessLogSettings.Format,
		}
	}
	p.CreatedDate = *resp.CreatedDate
	p.LastUpdatedDate = *resp.LastUpdatedDate
	p.MethodSettings = make(map[string]MethodSetting)
	for k, v := range resp.MethodSettings {
		p.MethodSettings[k] = MethodSetting{
			CacheDataEncrypted:                     *v.CacheDataEncrypted,
			CacheTTLInSeconds:                      *v.CacheTtlInSeconds,
			CachingEnabled:                         *v.CachingEnabled,
			DataTraceEnabled:                       *v.DataTraceEnabled,
			LoggingLevel:                           *v.LoggingLevel,
			MetricsEnabled:                         *v.MetricsEnabled,
			RequireAuthorizationForCacheControl:    *v.RequireAuthorizationForCacheControl,
			ThrottlingBurstLimit:                   *v.ThrottlingBurstLimit,
			ThrottlingRateLimit:                    *v.ThrottlingRateLimit,
			UnauthorizedCacheControlHeaderStrategy: string(v.UnauthorizedCacheControlHeaderStrategy),
		}
	}
	p.WebACLARN = resp.WebAclArn

	return nil
}

// Delete removes a deployment.
func (p *APIGatewayStage) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteStageRequest(&apigateway.DeleteStageInput{
		RestApiId: aws.String(p.RestAPIID),
		StageName: aws.String(p.StageName),
	})
	_, err = req.Send(ctx)
	return err
}

// Update triggers a new deployment.
// There is no concept of "updating" a deployment so it is identical to
// creating a new one.
func (p *APIGatewayStage) Update(ctx context.Context, r *resource.UpdateRequest) error {
	prev := r.Previous.(*APIGatewayStage)

	if prev.Region != p.Region ||
		prev.RestAPIID != p.RestAPIID ||
		prev.StageName != p.StageName {
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
		apigatewaypatch.Field{Name: "CacheClusterEnabled", Path: "/cacheClusterEnabled"},
		apigatewaypatch.Field{Name: "CacheClusterSize", Path: "/cacheClusterSize"},
		apigatewaypatch.Field{Name: "DeploymentID", Path: "/deploymentId"},
		apigatewaypatch.Field{Name: "Description", Path: "/description"},
		apigatewaypatch.Field{Name: "DocumentationVersion", Path: "/documentationVersion"},
		apigatewaypatch.Field{Name: "Tags", Path: "/tags"},
		apigatewaypatch.Field{Name: "TracingEnabled", Path: "/tracingEnabled"},
		apigatewaypatch.Field{Name: "Variables", Path: "/variables"},
		// TODO: Handle canary settings
	)
	if err != nil {
		return backoff.Permanent(errors.Wrap(err, "resolve patch operations"))
	}

	if len(ops) == 0 {
		return nil
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.UpdateStageRequest(&apigateway.UpdateStageInput{
		PatchOperations: ops,
		RestApiId:       aws.String(p.RestAPIID),
		StageName:       aws.String(p.StageName),
	})
	resp, err := req.Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == apigateway.ErrCodeBadRequestException {
				return backoff.Permanent(err)
			}
		}
		return err
	}

	if resp.AccessLogSettings != nil {
		p.AccessLogSettings = &AccessLogSettings{
			DestinationARN: resp.AccessLogSettings.DestinationArn,
			Format:         resp.AccessLogSettings.Format,
		}
	}
	p.CreatedDate = *resp.CreatedDate
	p.LastUpdatedDate = *resp.LastUpdatedDate
	p.MethodSettings = make(map[string]MethodSetting)
	for k, v := range resp.MethodSettings {
		p.MethodSettings[k] = MethodSetting{
			CacheDataEncrypted:                     *v.CacheDataEncrypted,
			CacheTTLInSeconds:                      *v.CacheTtlInSeconds,
			CachingEnabled:                         *v.CachingEnabled,
			DataTraceEnabled:                       *v.DataTraceEnabled,
			LoggingLevel:                           *v.LoggingLevel,
			MetricsEnabled:                         *v.MetricsEnabled,
			RequireAuthorizationForCacheControl:    *v.RequireAuthorizationForCacheControl,
			ThrottlingBurstLimit:                   *v.ThrottlingBurstLimit,
			ThrottlingRateLimit:                    *v.ThrottlingRateLimit,
			UnauthorizedCacheControlHeaderStrategy: string(v.UnauthorizedCacheControlHeaderStrategy),
		}
	}
	p.WebACLARN = resp.WebAclArn

	return nil
}
