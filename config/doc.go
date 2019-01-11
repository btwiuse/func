// Package config provides types for project configuration and loading config
// files from disk.
//
// The files are loaded using the Loader, and then can be marshalled into json.
// The resulting json should be unmarshalled into a *hclpack.Body, which can
// then be decoded into the Root config.
//
// A typical config file may look something like this:
//
//  project "example" {}
//
//  resource "aws_lambda_function" "func" {
//    source = "./src"          # source code location
//
//    handler = "index.handler" # attributes passed
//    memory  = 512             # to resource
//  }
//
// If a resource specifies a source attribute, the source files from the
// directory are collected and hashed. The hash is added as a digest to the
// resource. To get a list of the original source files, the Loader can be
// queried.
//
// Except for the source, the entire body of a resource is specific to the
// resource type, set by the first label.
package config
