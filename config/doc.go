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
// Source code
//
// If a resource specifies a source attribute, the source files from the
// directory are compressed into a source archive using the Compressor set on
// the Loader.
//
// The source attribute is replaced with a source info block:
//
//  resource "aws_lambda_function" "func" {
//    source ".tar.gz" {   # source extension
//      sha = "b5bb9d8..." # sha256 hash of file contents
//      md5 = "ERZsvZy..." # md5 hash of source archive
//      len = 127          # source archive size in bytes
//    }
//
//    handler = "index.handler" # attributes passed
//    memory  = 512             # to resource
//  }
//
// Source() can be used to get a pointer to the source archive when source code
// is needed.
//
// Except for the source, the entire body of a resource is specific to the
// resource type, set by the first label.
package config
