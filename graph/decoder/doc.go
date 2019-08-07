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
//
//       sub {
//           value = 123
//       }
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
//       Name    string   `func:"input"`
//       Age     *int     `func:"input"`
//       Address *Address `func:"input"`
//
//       // Outputs
//       Nickname string `func:"output" name="nick"`
//   }
//
//   type Address struct {
//       StreetName string  `func:"input"`
//       Country    *string `func:"input"`
//   }
//
// The value in the tag is matched to the attribute in hcl. This means `nick`
// is the word to use in the example above. By convention, tag names should be
// lower_snake_case.
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
//      # A dependency for the iam.role.arn -> lambda.logic.role is created.
//      role = iam.role.arn
//   }
//
//   resource "apigateway" "integration" {
//      # A dependency from the iam.role and lambda.logic is created.
//        When both values have been resolved at runtime, the concatenated
//        value is inserted as uri.
//      uri = "${iam.role.arn}-${lambda.logic.arn}
//   }
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
// Inputs are followed until an output is discovered, at which point a
// dependency is created. Inputs that can be statically resolved are resolved
// in place to allows resources to execute concurrently.
//
// Nested blocks
//
// Nested blocks do currently NOT support dynamic inputs. This is a known limitation.
//
// The following hypothetical example would result in an error:
//
//   resource "iam" "role" {
//   }
//
//   resource "lambda" logic" {
//     value = iam.role.arn            # Argument level reference is ok
//
//     environment {
//       nested_value = iam.role.arn   # Cannot add dynamic dependency inside of a block
//     }
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
