package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
)

// A Root is the root structure of a project's configuration, including all
// resources that are part of the project.
type Root struct {
	Resources []Resource `hcl:"resource,block"`
}

// Resource is a user specified resource specification.
type Resource struct {
	// Name is a unique name (within the same kind) for the resource.
	Name string `hcl:"name,label"`

	// Type specifies what type of resource this is.
	//
	// The type defines how the Config is decoded.
	Type string `hcl:"type"`

	// Config is a configuration body for the resource.
	//
	// The contents will depend on the resource type.
	Config hcl.Body `hcl:",remain"`

	// SourceDigest contains information about the attached source code. The
	// field is nil if the resource has no source.
	Source string `hcl:"source,optional"`
}

// SourceInfo contains information about the resource source code.
type SourceInfo struct {
	Key string // Unique key for source based on content digest.
	MD5 string // Base64 encoded MD5 checksum of compressed source.
	Len int    // Source archive size in Bytes.
}

// EncodeToString encodes the source info to a string.
func (s SourceInfo) EncodeToString() string {
	return fmt.Sprintf("%x:%s:%s", s.Len, s.MD5, s.Key)
}

// DecodeSourceString decodes a source string encoded by EncodeToString().
func DecodeSourceString(str string) (SourceInfo, error) {
	var src SourceInfo
	parts := strings.Split(str, ":")
	if len(parts) != 3 {
		return src, fmt.Errorf("string must contain 3 parts separated by ':'")
	}
	l, err := strconv.ParseInt(parts[0], 16, 32)
	if err != nil {
		return src, errors.Wrapf(err, "extract length from %q", parts[0])
	}
	src.Len = int(l)
	src.MD5 = parts[1]
	src.Key = parts[2]
	if src.MD5 == "" {
		return src, errors.New("md5 not set")
	}
	if src.Key == "" {
		return src, errors.New("key not set")
	}
	return src, nil
}
