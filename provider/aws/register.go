package aws

import (
	"github.com/func/func/resource"
)

type registry interface {
	Register(resource.Definition)
}

// Register adds all supported AWS resources to the registry.
func Register(reg registry) {
	reg.Register(&LambdaFunction{})
	reg.Register(&IAMPolicyDocument{})
	reg.Register(&IAMRole{})
	reg.Register(&IAMPolicy{})
	reg.Register(&IAMRolePolicy{})
	reg.Register(&IAMRolePolicyAttachment{})
	reg.Register(&APIGatewayRestAPI{})
	reg.Register(&APIGatewayResource{})
}
