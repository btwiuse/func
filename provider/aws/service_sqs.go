package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/func/func/resource"
)

type sqsService struct {
	client sqsiface.ClientAPI
}

// service returns a SQS API Client. If client was set, it is returned.
func (p *sqsService) service(auth resource.AuthProvider, region string) (sqsiface.ClientAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return sqs.New(cfg), nil
}
