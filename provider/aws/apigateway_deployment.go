package aws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// APIGatewayDeployment provides a Serverless REST API.
type APIGatewayDeployment struct {
	// Inputs

	// Enables a cache cluster for the Stage resource specified in the input.
	CacheClusterEnabled *bool `func:"input"`

	// Specifies the cache cluster size for the Stage resource specified in the
	// input, if a cache cluster is enabled.
	CacheClusterSize *string `func:"input" validate:"oneof=0.5 1.6 6.1 13.5 28.4 58.2 118 237"`

	// The input configuration for the canary deployment when the deployment is
	// a canary release deployment.
	CanarySettings *APIGatewayDeploymentCanarySettings `func:"input"`

	// The description for the Deployment resource to create.
	Description *string `func:"input"`

	// The region the API Gateway is deployed to.
	Region string `func:"input,required"`

	// The string identifier of the associated RestApi.
	RestAPIID *string `func:"input,required" name:"rest_api_id"`

	// The description of the Stage resource for the Deployment resource to
	// create.
	StageDescription *string `func:"input"`

	// The name of the Stage resource for the Deployment resource to create.
	StageName *string `func:"input"`

	// Specifies whether active tracing with X-ray is enabled for the Stage.
	TracingEnabled *bool `func:"input"`

	// A map that defines the stage variables for the Stage resource that is
	// associated with the new deployment. Variable names can have alphanumeric
	// and underscore characters, and the values must match
	// `[A-Za-z0-9-._~:/?#&=,]+`.
	Variables map[string]string `func:"input"`

	// ChangeTrigger causes a new deployment to be executed when the value has
	// changed, even if other inputs have not changed.
	ChangeTrigger string `func:"input,required"`

	// Outputs

	// A summary of the RestApi at the date and time that the deployment resource
	// was created.
	APISummary map[string]map[string]APIGatewayMethodSnapshot `func:"output"`

	// The date and time that the deployment resource was created.
	CreatedDate *time.Time `func:"output"`

	// The identifier for the deployment resource.
	ID *string `func:"output"`

	apigatewayService
}

// APIGatewayDeploymentCanarySettings contains settings for canary deployment,
// passed as input to APIGatewayDeployment.
type APIGatewayDeploymentCanarySettings struct {
	// The percentage (0.0-100.0) of traffic routed to the canary deployment.
	PercentTraffic *float64 `func:"input" validate:"gte=0,lte=100"`

	// A stage variable overrides used for the canary release deployment. They
	// can override existing stage variables or add new stage variables for the
	// canary release deployment. These stage variables are represented as a
	// string-to-string map between stage variable names and their values.
	StageVariableOverrides map[string]string `func:"input"`

	// A Boolean flag to indicate whether the canary release deployment uses
	// the stage cache or not.
	UseStageCache *bool `func:"input"`
}

// APIGatewayMethodSnapshot contains a snapshot of a deployed method. It is
// provided as an output from APIGatewayDeployment.
type APIGatewayMethodSnapshot struct {
	// Specifies whether the method requires a valid ApiKey.
	APIKeyRequired bool `func:"output"`

	// The method's authorization type. Valid values are `NONE` for open
	// access, `AWS_IAM` for using AWS IAM permissions, `CUSTOM` for using a
	// custom authorizer, or `COGNITO_USER_POOLS` for using a Cognito user
	// pool.
	AuthorizationType string `func:"output"`
}

// Create creates a new deployment.
func (p *APIGatewayDeployment) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.CreateDeploymentInput{
		Description:      p.Description,
		RestApiId:        p.RestAPIID,
		StageDescription: p.StageDescription,
		StageName:        p.StageName,
		TracingEnabled:   p.TracingEnabled,
		Variables:        p.Variables,
	}

	if p.CacheClusterSize != nil {
		input.CacheClusterSize = apigateway.CacheClusterSize(*p.CacheClusterSize)
	}
	if p.CanarySettings != nil {
		input.CanarySettings = &apigateway.DeploymentCanarySettings{
			PercentTraffic:         p.CanarySettings.PercentTraffic,
			StageVariableOverrides: p.CanarySettings.StageVariableOverrides,
			UseStageCache:          p.CanarySettings.UseStageCache,
		}
	}

	sha := sha256.New()
	if _, err := sha.Write([]byte(p.ChangeTrigger)); err != nil {
		return err
	}
	if input.Variables == nil {
		input.Variables = make(map[string]string)
	}
	input.Variables["func_change_trigger_hash"] = hex.EncodeToString(sha.Sum(nil))

	req := svc.CreateDeploymentRequest(input)
	resp, err := req.Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == apigateway.ErrCodeBadRequestException {
				return backoff.Permanent(err)
			}
		}
		return err
	}

	p.APISummary = make(map[string]map[string]APIGatewayMethodSnapshot, len(resp.ApiSummary))
	for k, v := range resp.ApiSummary {
		p.APISummary[k] = make(map[string]APIGatewayMethodSnapshot)
		for kk, vv := range v {
			p.APISummary[k][kk] = APIGatewayMethodSnapshot{
				APIKeyRequired:    *vv.ApiKeyRequired,
				AuthorizationType: *vv.AuthorizationType,
			}
		}
	}

	p.CreatedDate = resp.CreatedDate
	p.ID = resp.Id

	return nil
}

// Delete removes a deployment.
func (p *APIGatewayDeployment) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteDeploymentRequest(&apigateway.DeleteDeploymentInput{
		RestApiId:    p.RestAPIID,
		DeploymentId: p.ID,
	})
	_, err = req.Send(ctx)
	return err
}

// Update triggers a new deployment.
// There is no concept of "updating" a deployment so it is identical to
// creating a new one.
func (p *APIGatewayDeployment) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Update is the same as create
	return p.Create(ctx, r.CreateRequest())
}
