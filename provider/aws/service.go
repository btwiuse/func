package aws

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

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

func handlePutError(err error) error {
	if err == nil {
		return nil
	}
	if aerr, ok := err.(awserr.RequestFailure); ok {
		if aerr.StatusCode() == http.StatusTooManyRequests {
			return err
		}
		if aerr.StatusCode() >= 400 && aerr.StatusCode() < 500 {
			return backoff.Permanent(err)
		}
		return err
	}
	return err
}

func handleDelError(err error) error {
	if err == nil {
		return nil
	}
	if aerr, ok := err.(awserr.RequestFailure); ok {
		if aerr.StatusCode() == 404 {
			// Already deleted
			return nil
		}
		if aerr.StatusCode() == http.StatusTooManyRequests {
			return err
		}
		if aerr.StatusCode() >= 400 && aerr.StatusCode() < 500 {
			return backoff.Permanent(err)
		}
		return err
	}
	return err
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
