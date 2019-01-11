package config

import "github.com/hashicorp/hcl2/hcl"

// A Root is the root structure of a project's configuration, including all
// resources that are part of the project.
type Root struct {
	Project   Project    `hcl:"project,block"`
	Resources []Resource `hcl:"resource,block"`
}

// A Project is the root object of a func project.
type Project struct {
	// Name uniquely identifies a project.
	Name string `hcl:"name,label"`
}

// Resource is a user specified resource specification.
type Resource struct {
	// Type specifies what type of resource this is.
	Type string `hcl:"type,label"`

	// Name is a unique name (within the same kind) for the resource.
	Name string `hcl:"name,label"`

	// Config is a configuration body for the resource.
	//
	// The contents will depend on the resource type.
	Config hcl.Body `hcl:",remain"`

	// SourceDigest is a hash digest computed from the contents of the source code on
	// disk. The hash is computed from all the individual source files, before
	// any transformations or build steps.
	//
	// The digest is deterministic, as long as the contents of the files
	// doesn't change. The name of the file does not impact the hash, except if
	// it changes the ordering.
	//
	// The field is an empty string if the resource has no source.
	SourceDigest *string `hcl:"digest"`
}
