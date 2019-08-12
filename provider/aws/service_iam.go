package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/iamiface"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

type iamService struct {
	client iamiface.ClientAPI
}

// service returns an IAM API Client. If client was set, it is returned.
func (p *iamService) service(auth resource.AuthProvider, region *string) (iamiface.ClientAPI, error) {
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
		return nil, backoff.Permanent(err)
	}
	return iam.New(cfg), nil
}
