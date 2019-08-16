package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/func/func/resource"
)

type dynamoDBService struct {
	client dynamodbiface.ClientAPI
}

// service returns a DynamoDB API Client. If client was set, it is returned.
func (p *dynamoDBService) service(auth resource.AuthProvider, region string) (dynamodbiface.ClientAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	cfg, err := awsConfig(auth, region)
	if err != nil {
		return nil, err
	}
	return dynamodb.New(cfg), nil
}
