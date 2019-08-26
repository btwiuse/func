// Package hcldecoder provides a decoder for hcl2 resources that define a
// graph.
//
// Such a config may look something like this:
//
//   resource "person_a" {
//       type = "person"
//       name = "alice"
//
//       pet {
//           name = "fido"
//       }
//   }
//
//   resource "person_b" {
//       type = "person"
//       name = "bob"
//
//       friends = [person_a.name]
//   }
//
// The type determines how to decode the remaining configuration. This type is
// matched to return a resource schema.
//
// The package will return hcl.Diagnostics for any errors, which should always
// be displayed to the user. If the diagnostics contain errors, the graph may
// be partially populated but should not be considered correct or complete.
//
// Dependencies
//
// A resource may declare a dependency to another resource:
//   input = other.output
//
// In addition, dependencies can be combined expressions, consisting of
// literals and references:
//   input = "hello_${other.name}!"
//
// Dependencies that can be statically resolved are not created. For example,
// if resource b refers an input in resource a, the value will directly be
// populated in b, without a dependency on a:
//
//   resource "friend" {
//       type = "person"
//       name = "bob"
//   }
//
//   resource "welcome" {
//       type     = "greeter"
//       greeting = "Hello, ${friend.name}!" # Statically resolved to "Hello, bob!"
//   }
//
// This applies to partially populating expressions as well. The parts that can
// be statically resolved will replace referenced values in the expression,
// while dependencies for the remaining fields are still created:
//
//   resource "friend" {
//       type = "person"
//       name = "bob"
//   }
//
//   resource "greeter" {
//       type = "greeting" # Has an output greeting
//   }
//
//   resource "greet" {
//       type     = "greeter"
//       greeting = "${greeter.greeting}, ${friend.name}!" # Resolved to "${greeter.greeting}, bob!"
//   }
//
// Any references to outputs of resources will create dependencies, as the
// values are only known after the resource provides output values. These will
// create dependencies in the graph.
//
// Parent references
//
// Whenever the source config contains a reference to another resource, a
// parent reference is added to the resource in the graph. When the reference
// can be statically resolved, a dependency is not added, but the parent
// reference is still kept.
//
// Source
//
// Source code that is set on the resource will be decoded and returned. The
// source key is added to the resource itself, while the original decoded
// source info is returned (to ensure source code exists and request it if
// needed).
//
// The source code is not provided by the user directly, failing to decode it
// likely is a bug in the encoder or decoder.
package hcldecoder
