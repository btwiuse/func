package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/iamiface"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/stsiface"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

type apigatewayService struct {
	client apigatewayiface.APIGatewayAPI
}

// service returns an APIGateway API Client. If client was set, it is returned.
func (p *apigatewayService) service(auth resource.AuthProvider, region string) (apigatewayiface.APIGatewayAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return apigateway.New(cfg), nil
}

type iamService struct {
	client iamiface.IAMAPI
}

// service returns an IAM API Client. If client was set, it is returned.
func (p *iamService) service(auth resource.AuthProvider, region *string) (iamiface.IAMAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	var reg string
	if region != nil {
		reg = *region
	} else {
		reg = defaultRegion()
	}
	cfg, err := awsConfig(auth, reg)
	if err != nil {
		return nil, err
	}
	return iam.New(cfg), nil
}

type lambdaService struct {
	client lambdaiface.LambdaAPI
}

// service returns a Lambda API Client. If client was set, it is returned.
func (p *lambdaService) service(auth resource.AuthProvider, region string) (lambdaiface.LambdaAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return lambda.New(cfg), nil
}

type stsService struct {
	client stsiface.STSAPI
}

// service returns an STS API Client. If client was set, it is returned.
func (p *stsService) service(auth resource.AuthProvider, region *string) (stsiface.STSAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	var reg string
	if region != nil {
		reg = *region
	} else {
		reg = defaultRegion()
	}
	cfg, err := awsConfig(auth, reg)
	if err != nil {
		return nil, err
	}
	return sts.New(cfg), nil
}

// ---

func awsConfig(auth resource.AuthProvider, region string) (aws.Config, error) {
	cfg := defaults.Config()
	creds, err := auth.AWS()
	if err != nil {
		return aws.Config{}, errors.Wrap(err, "get credentials")
	}
	cfg.Credentials = creds
	cfg.Region = region
	return cfg, nil
}

// defaultRegion determines the default region to use based on:
//
//  - From AWS_DEFAULT_REGION environment variable.
//  - From region in ~/.aws/credentials.
//  - If neither is set, us-east-1 is used.
func defaultRegion() string {
	const fallback = endpoints.UsEast1RegionID
	var cfgs external.Configs
	cfgs, err := cfgs.AppendFromLoaders(external.DefaultConfigLoaders)
	if err != nil {
		return fallback
	}
	cfg, err := cfgs.ResolveAWSConfig([]external.AWSConfigResolver{
		external.ResolveRegion,
	})
	if err != nil {
		return fallback
	}
	if cfg.Region == "" {
		// No AWS config available
		return fallback
	}
	return cfg.Region
}
