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

// If a mock service is set, the corresponding mock api client is returned when
// a client is requested.
var (
	mockAPIGatewayService apigatewayiface.APIGatewayAPI
	mockIAMService        iamiface.IAMAPI
	mockLambdaService     lambdaiface.LambdaAPI
	mockSTSService        stsiface.STSAPI
)

func apigatewayService(auth resource.AuthProvider, region string) (apigatewayiface.APIGatewayAPI, error) {
	if mockAPIGatewayService != nil {
		return mockAPIGatewayService, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return apigateway.New(cfg), nil
}

func iamService(auth resource.AuthProvider, region *string) (iamiface.IAMAPI, error) {
	if mockIAMService != nil {
		return mockIAMService, nil
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

func lambdaService(auth resource.AuthProvider, region string) (lambdaiface.LambdaAPI, error) {
	if mockLambdaService != nil {
		return mockLambdaService, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return lambda.New(cfg), nil
}

func stsService(auth resource.AuthProvider, region *string) (stsiface.STSAPI, error) {
	if mockSTSService != nil {
		return mockSTSService, nil
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
	cfg.Region = region
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
