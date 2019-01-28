// Package decoder provides decoding of a hcl body to a graph.
//
// Project
//
// A project node is added to the graph when a project block is encountered:
//
//   project "example" {
//   }
//
// Resources
//
// A Resource is matched based on the first label set in the resource block.
// The string is matched to a resource from the DecodeContext. For example, the
// following would parse the config as the resource that's registered as
// aws_lambda_function:
//
//   resource "aws_lambda_function" "test" {
//       name = "example"
//   }
//
// The second label ("test") is used when referring to resources from other
// resources.
//
// Decoding resource config
//
// Struct tags from the resource are used to describe the inputs/outputs:
//
//   type MyResource struct {
//       // Inputs
//       Name string `input:"input"`
//       Age  *int   `input:"age"`
//
//       // Outputs
//       Nickname string `output:"nick"`
//   }
//
// The value in the tag is matched to the attribute in hcl. This means `nick`
// is the word to use in the example above. By convention, tag names should be
// lower_snake_case.
//
// If an input is set on a pointer, the input is optional.
//
// References
//
// A Resource may define references to other resources, creating a dependency
// on the resource. The available output types are defined in the Resource
// definition, passed in the decode context.
//
//   resource "iam" "role" {
//      role_name = "example"
//   }
//
//   resource "lambda" "logic" {
//      role = iam.role.arn        # A dependency for the iam.role.arn -> lambda.logic.role is created.
//   }
//
//
//
// In case a resource references an attribute which is an input, the value is
// passed to the child resource.
//
//   resource "iam" "role" {
//      role_name = "example"
//   }
//
//   resource "lambda" "logic" {
//      role = iam.role.role_name  # No dependency is created, a lambda resource is added with the role "example" set.
//   }
//
// Source
//
// If the resource has a source block, a source dependency is added to the
// resource. The source block itself is not decoded into the resource in any
// way.
//
//   resource "lambda" "func" {
//       source ".tar.gz" {
//           sha = "..."
//           md5 = "..."
//           len = 123
//       }
//   }
package decoder
