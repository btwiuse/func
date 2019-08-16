package aws

import (
	"github.com/func/func/resource"
)

type registry interface {
	Register(typename string, def resource.Definition)
}

type validator interface {
	Add(name string, validate func(interface{}, string) error)
}

// Register adds all supported AWS resources to the registry.
func Register(reg registry) {
	reg.Register("aws_apigateway_deployment", &APIGatewayDeployment{})
	reg.Register("aws_apigateway_integration", &APIGatewayIntegration{})
	reg.Register("aws_apigateway_method", &APIGatewayMethod{})
	reg.Register("aws_apigateway_resource", &APIGatewayResource{})
	reg.Register("aws_apigateway_rest_api", &APIGatewayRestAPI{})
	reg.Register("aws_apigateway_stage", &APIGatewayStage{})
	reg.Register("aws_iam_policy", &IAMPolicy{})
	reg.Register("aws_iam_policy_document", &IAMPolicyDocument{})
	reg.Register("aws_iam_role", &IAMRole{})
	reg.Register("aws_iam_role_policy", &IAMRolePolicy{})
	reg.Register("aws_iam_role_policy_attachment", &IAMRolePolicyAttachment{})
	reg.Register("aws_lambda_function", &LambdaFunction{})
	reg.Register("aws_lambda_invoke_permission", &LambdaInvokePermission{})
	reg.Register("aws_sts_caller_identity", &STSCallerIdentity{})
}
