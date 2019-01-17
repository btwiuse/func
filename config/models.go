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

	// SourceDigest contains information about the attached source code. The
	// field is nil if the resource has no source.
	Source *SourceInfo `hcl:"source,block"`
}

type SourceInfo struct {
	Ext string `hcl:"ext,label"` // Source archive file extension.
	SHA string `hcl:"sha"`       // Hex encoded.
	MD5 string `hcl:"md5"`       // Base64 encoded.
	Len int    `hcl:"len"`       // Source archive size in Bytes.
}
