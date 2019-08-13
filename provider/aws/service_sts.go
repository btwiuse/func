package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/stsiface"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

type stsService struct {
	client stsiface.ClientAPI
}

// service returns an STS API Client. If client was set, it is returned.
func (p *stsService) service(auth resource.AuthProvider, region *string) (stsiface.ClientAPI, error) {
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
	return sts.New(cfg), nil
}
