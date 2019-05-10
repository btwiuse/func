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
	CacheClusterEnabled *bool `input:"cache_cluster_enabled"`

	// Specifies the cache cluster size for the Stage resource specified in the
	// input, if a cache cluster is enabled.
	//
	// Value must be one of:
	// - "0.5"
	// - "1.6"
	// - "6.1"
	// - "13.5"
	// - "28.4"
	// - "58.2"
	// - "118"
	// - "237"
	CacheClusterSize *string `input:"cache_cluster_size"`

	// The input configuration for the canary deployment when the deployment is
	// a canary release deployment.
	CanarySettings *APIGatewayCanarySettings `input:"canary_settings"`

	// The identifier of the Deployment resource for the Stage resource.
	DeploymentID string `input:"deployment_id"`

	// The description of the Stage resource.
	Description *string `input:"description"`

	// The version of the associated API documentation.
	DocumentationVersion *string `input:"deployment_version"`

	// The region the API Gateway is deployed to.
	Region string `input:"region"`

	// The string identifier of the associated RestApi.
	RestAPIID string `input:"rest_api_id"`

	// The name for the Stage resource.
	StageName string `input:"stage_name"`

	// The key-value map of strings. The valid character set is `[a-zA-Z+-=._:/]`.
	// The tag key can be up to 128 characters and must not start with aws:. The
	// tag value can be up to 256 characters.
	Tags *map[string]string `input:"tags"`

	// Specifies whether active tracing with X-ray is enabled for the Stage.
	TracingEnabled *bool `input:"tracing_enabled"`

	// A map that defines the stage variables for the new Stage resource. Variable
	// names can have alphanumeric and underscore characters, and the values must
	// match `[A-Za-z0-9-._~:/?#&=,]+`.
	Variables *map[string]string `input:"variables"`

	// Outputs

	// Settings for logging access in this stage.
	AccessLogSettings *AccessLogSettings `output:"access_log_settings"`

	// The status of the cache cluster for the stage, if enabled.
	// Value will be one of:
	// - `CREATE_IN_PROGRESS`
	// - `AVAILABLE`
	// - `DELETE_IN_PROGRESS`
	// - `NOT_AVAILABLE`
	// - `FLUSH_IN_PROGRESS`
	CacheClusterStatus string `output:"cache_cluster_status"`

	// The identifier of a client certificate for an API stage.
	ClientCertificateID string `output:"client_certificate_id"`

	// The timestamp when the stage was created.
	CreatedDate time.Time `output:"created_date"`

	// The timestamp when the stage last updated.
	LastUpdatedDate time.Time `output:"last_updated_date"`

	// A map that defines the method settings for a Stage resource. Keys (designated
	// as /{method_setting_key below) are method paths defined as {resource_path}/{http_method}
	// for an individual method override, or /\*/\* for overriding all methods in
	// the stage.
	MethodSettings map[string]MethodSetting `output:"method_settings"`

	// The ARN of the WebAcl associated with the Stage.
	WebACLARN *string `output:"web_acl_arn"`

	apigatewayService
}

// APIGatewayCanarySettings contains settings for canary deployment.
type APIGatewayCanarySettings struct {
	// The ID of the canary deployment.
	DeploymentID *string `input:"deployment_id"`

	// The percent (0-100) of traffic diverted to a canary deployment.
	PercentTraffic *float64 `input:"percent_traffic"`

	// Stage variables overridden for a canary release deployment, including new
	// stage variables introduced in the canary. These stage variables are represented
	// as a string-to-string map between stage variable names and their values.
	StageVariableOverrides *map[string]string `input:"stage_variable_overrides"`

	// A Boolean flag to indicate whether the canary deployment uses the stage cache
	// or not.
	UseStageCache *bool `input:"use_stage_cache"`
}

// AccessLogSettings contains settings for an APIGateway deployment stage.
type AccessLogSettings struct {
	// The ARN of the CloudWatch Logs log group to receive access logs.
	DestinationArn *string `output:"destination_arn"`

	// A single line format of the access logs of data, as specified by
	// selected `$context` variables.
	// The format must include at least `$context.requestId`.
	Format *string `output:"format"`
}

// MethodSetting contains settings for a method in an APIGateway deployment stage.
type MethodSetting struct { // nolint: maligned
	// Specifies whether the cached responses are encrypted. The PATCH path for
	// this setting is /{method_setting_key}/caching/dataEncrypted, and the value
	// is a Boolean.
	CacheDataEncrypted bool `output:"cache_data_encrypted"`

	// Specifies the time to live (TTL), in seconds, for cached responses. The higher
	// the TTL, the longer the response will be cached. The PATCH path for this
	// setting is /{method_setting_key}/caching/ttlInSeconds, and the value is an
	// integer.
	CacheTTLInSeconds int64 `output:"cache_ttl_seconds"`

	// Specifies whether responses should be cached and returned for requests. A
	// cache cluster must be enabled on the stage for responses to be cached. The
	// PATCH path for this setting is /{method_setting_key}/caching/enabled, and
	// the value is a Boolean.
	CachingEnabled bool `output:"caching_enabled"`

	// Specifies whether data trace logging is enabled for this method, which affects
	// the log entries pushed to Amazon CloudWatch Logs. The PATCH path for this
	// setting is /{method_setting_key}/logging/dataTrace, and the value is a Boolean.
	DataTraceEnabled bool `output:"data_trace_enabled"`

	// Specifies the logging level for this method, which affects the log entries
	// pushed to Amazon CloudWatch Logs. The PATCH path for this setting is /{method_setting_key}/logging/loglevel,
	// and the available levels are OFF, ERROR, and INFO.
	LoggingLevel string `output:"logging_level"`

	// Specifies whether Amazon CloudWatch metrics are enabled for this method.
	// The PATCH path for this setting is /{method_setting_key}/metrics/enabled,
	// and the value is a Boolean.
	MetricsEnabled bool `output:"metrics_enabled"`

	// Specifies whether authorization is required for a cache invalidation request.
	// The PATCH path for this setting is /{method_setting_key}/caching/requireAuthorizationForCacheControl,
	// and the value is a Boolean.
	RequireAuthorizationForCacheControl bool `output:"require_authorization_for_cache_control"`

	// Specifies the throttling burst limit. The PATCH path for this setting is
	// /{method_setting_key}/throttling/burstLimit, and the value is an integer.
	ThrottlingBurstLimit int64 `output:"throttling_burst_limit"`

	// Specifies the throttling rate limit. The PATCH path for this setting is /{method_setting_key}/throttling/rateLimit,
	// and the value is a double.
	ThrottlingRateLimit float64 `output:"throttling_rate_limit"`

	// Specifies how to handle unauthorized requests for cache invalidation.
	// Available values are:
	// - `FAIL_WITH_403`
	// - `SUCCEED_WITH_RESPONSE_HEADER`
	// - `SUCCEED_WITHOUT_RESPONSE_HEADER`
	UnauthorizedCacheControlHeaderStrategy string `output:"unauthorized_cache_control_header_strategy"`
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
	if p.Tags != nil {
		input.Tags = make(map[string]string)
		for k, v := range *p.Tags {
			input.Tags[k] = v
		}
	}
	if p.Variables != nil {
		input.Variables = make(map[string]string)
		for k, v := range *p.Variables {
			input.Variables[k] = v
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

	p.setFromResp(resp)

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

	p.setFromResp(resp)

	return nil
}

func (p *APIGatewayStage) setFromResp(resp *apigateway.UpdateStageOutput) {
	if resp.AccessLogSettings != nil {
		p.AccessLogSettings = &AccessLogSettings{
			DestinationArn: resp.AccessLogSettings.DestinationArn,
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
}
