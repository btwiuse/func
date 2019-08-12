package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/lambdaiface"
	"github.com/func/func/resource"
)

type lambdaService struct {
	client lambdaiface.ClientAPI
}

// service returns a Lambda API Client. If client was set, it is returned.
func (p *lambdaService) service(auth resource.AuthProvider, region string) (lambdaiface.ClientAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return lambda.New(cfg), nil
}
