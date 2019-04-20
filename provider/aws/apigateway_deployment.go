//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct APIGatewayDeployment --template ../../tools/structdoc/template.txt --data type=aws_apigateway_deployment --output ../../docs/resources/aws/apigateway_deployment.md

package aws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/apigatewayiface"
	"github.com/cenkalti/backoff"
	"github.com/func/func/provider/aws/internal/config"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// APIGatewayDeployment provides a Serverless REST API.
type APIGatewayDeployment struct {
	// Inputs

	// Enables a cache cluster for the Stage resource specified in the input.
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

	// The description for the Deployment resource to create.
	Description *string `input:"description"`

	// The string identifier of the associated RestApi.
	RestAPIID string `input:"rest_api_id"`

	// The description of the Stage resource for the Deployment resource to
	// create.
	StageDescription *string `input:"state_description"`

	// The name of the Stage resource for the Deployment resource to create.
	StageName string `input:"stage_name"`

	// Specifies whether active tracing with X-ray is enabled for the Stage.
	TracingEnabled *bool `input:"tracing_enabled"`

	// A map that defines the stage variables for the Stage resource that is
	// associated with the new deployment. Variable names can have alphanumeric
	// and underscore characters, and the values must match
	// `[A-Za-z0-9-._~:/?#&=,]+`.
	Variables *map[string]string `input:"variables"`

	Region string `input:"region"`

	// ChangeTrigger causes a new deployment to be executed when the value has
	// changed, even if other inputs have not changed.
	ChangeTrigger string `input:"change_trigger"`

	// Outputs

	// A summary of the RestApi at the date and time that the deployment resource
	// was created.
	APISummary map[string]map[string]APIGatewayMethodSnapshot `output:"api_summary"`

	// The date and time that the deployment resource was created.
	CreatedDate time.Time `output:"created_date"`

	// The identifier for the deployment resource.
	ID string `locationName:"id" type:"string"`

	svc apigatewayiface.APIGatewayAPI
}

// APIGatewayCanarySettings contains settings for canary deployment, passed as
// input to APIGatewayDeployment.
type APIGatewayCanarySettings struct {
	// The percentage (0.0-100.0) of traffic routed to the canary deployment.
	PercentTraffic *float64 `input:"percent_traffic"`

	// A stage variable overrides used for the canary release deployment. They
	// can override existing stage variables or add new stage variables for the
	// canary release deployment. These stage variables are represented as a
	// string-to-string map between stage variable names and their values.
	StageVariableOverrides map[string]string `input:"state_variable_overrides"`

	// A Boolean flag to indicate whether the canary release deployment uses
	// the stage cache or not.
	UseStageCache *bool `input:"use_stage_cache"`
}

// APIGatewayMethodSnapshot contains a snapshot of a deployed method. It is
// provided as an output from APIGatewayDeployment.
type APIGatewayMethodSnapshot struct {
	// Specifies whether the method requires a valid ApiKey.
	APIKeyRequired bool `output:"api_key_required"`

	// The method's authorization type. Valid values are `NONE` for open
	// access, `AWS_IAM` for using AWS IAM permissions, `CUSTOM` for using a
	// custom authorizer, or `COGNITO_USER_POOLS` for using a Cognito user
	// pool.
	AuthorizationType string `output:"authorization_type"`
}

// Type returns the resource type of an apigateway deployment.
func (p *APIGatewayDeployment) Type() string { return "aws_apigateway_deployment" }

// Create creates a new deployment.
func (p *APIGatewayDeployment) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &apigateway.CreateDeploymentInput{
		Description:      p.Description,
		RestApiId:        aws.String(p.RestAPIID),
		StageDescription: p.StageDescription,
		StageName:        aws.String(p.StageName),
		TracingEnabled:   p.TracingEnabled,
		Variables:        make(map[string]string),
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
	if p.Variables != nil {
		for k, v := range *p.Variables {
			input.Variables[k] = v
		}
	}

	sha := sha256.New()
	if _, err := sha.Write([]byte(p.ChangeTrigger)); err != nil {
		return err
	}
	input.Variables["func_change_trigger_hash"] = hex.EncodeToString(sha.Sum(nil))

	req := svc.CreateDeploymentRequest(input)
	req.SetContext(ctx)
	resp, err := req.Send()
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

	p.CreatedDate = *resp.CreatedDate
	p.ID = *resp.Id

	// TODO: Provide way to log this
	fmt.Printf("---\nhttps://%s.execute-api.%s.amazonaws.com/%s\n---\n", p.RestAPIID, p.Region, p.StageName)

	return nil
}

// Delete removes a deployment.
func (p *APIGatewayDeployment) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteDeploymentRequest(&apigateway.DeleteDeploymentInput{
		RestApiId:    aws.String(p.RestAPIID),
		DeploymentId: aws.String(p.ID),
	})
	req.SetContext(ctx)
	_, err = req.Send()
	return err
}

// Update triggers a new deployment.
// There is no concept of "updating" a deployment so it is identical to
// creating a new one.
func (p *APIGatewayDeployment) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Update is the same as create
	return p.Create(ctx, r.CreateRequest())
}

func (p *APIGatewayDeployment) service(auth resource.AuthProvider) (apigatewayiface.APIGatewayAPI, error) {
	if p.svc == nil {
		cfg, err := config.WithRegion(auth, p.Region)
		if err != nil {
			return nil, errors.Wrap(err, "get aws config")
		}
		p.svc = apigateway.New(cfg)
	}
	return p.svc, nil
}
