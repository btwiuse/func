package aws

import (
	"github.com/func/func/resource"
)

type registry interface {
	Register(resource.Definition)
}

// Register adds all supported AWS resources to the registry.
func Register(reg registry) {
	reg.Register(&APIGatewayDeployment{})
	reg.Register(&APIGatewayIntegration{})
	reg.Register(&APIGatewayMethod{})
	reg.Register(&APIGatewayResource{})
	reg.Register(&APIGatewayRestAPI{})
	reg.Register(&IAMPolicyDocument{})
	reg.Register(&IAMPolicy{})
	reg.Register(&IAMRolePolicyAttachment{})
	reg.Register(&IAMRolePolicy{})
	reg.Register(&IAMRole{})
	reg.Register(&LambdaFunction{})
	reg.Register(&LambdaInvokePermission{})
	reg.Register(&STSCallerIdentity{})
}
