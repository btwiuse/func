package config

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// DefaultRegion returns an AWS configuration with the default region set.
//
// The default region is determined:
//  - From AWS_DEFAULT_REGION environment variable.
//  - From region in ~/.aws/credentials.
//  - If neither is set, us-east-1 is used.
func DefaultRegion(auth resource.AuthProvider) (aws.Config, error) {
	var cfgs external.Configs
	cfgs, err := cfgs.AppendFromLoaders(external.DefaultConfigLoaders)
	if err != nil {
		return aws.Config{}, err
	}
	cfg, err := cfgs.ResolveAWSConfig([]external.AWSConfigResolver{
		external.ResolveRegion,
	})
	if err != nil {
		return aws.Config{}, errors.Wrap(err, "resolve default config")
	}
	region := cfg.Region
	if region == "" {
		// No AWS config available
		region = "us-east-1"
	}
	return WithRegion(auth, region)
}

// WithRegion creates an AWS configuration with the given auth and region.
func WithRegion(auth resource.AuthProvider, region string) (aws.Config, error) {
	cfg := defaults.Config()
	creds, err := auth.AWS()
	if err != nil {
		return cfg, errors.Wrap(err, "get credentials")
	}
	cfg.Credentials = creds
	return cfg, nil
}
