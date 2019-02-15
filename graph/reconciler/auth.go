package reconciler

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

type tempLocalAuthProvider struct{}

func (p tempLocalAuthProvider) AWS() (aws.CredentialsProvider, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	return cfg.Credentials, nil
}
