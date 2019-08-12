package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/apigatewayiface"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

type apigatewayService struct {
	client apigatewayiface.ClientAPI
}

// service returns an APIGateway API Client. If client was set, it is returned.
func (p *apigatewayService) service(auth resource.AuthProvider, region string) (apigatewayiface.ClientAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, backoff.Permanent(err)
	}
	return apigateway.New(cfg), nil
}
