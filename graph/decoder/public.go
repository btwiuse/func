package decoder

import (
	"fmt"
	"strings"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
)

var rootSchema, _ = gohcl.ImpliedBodySchema(config.Root{})

// A ResourceRegistry is used for matching resource type names to resource
// implementations.
type ResourceRegistry interface {
	New(typename string) (resource.Definition, error)
	SuggestType(typename string) string
}

// DecodeContext is the context to use when decoding.
//
// For now, only the resource names can be provided.
type DecodeContext struct {
	Resources ResourceRegistry
}

// DecodeBody decodes a given raw configuration body into the target graph.
//
// Dependencies between resources are created as required and added to the
// graph. If expressions can be statically resolved, either directly or by
// following dependencies, they are not added as dependencies to the graph.
//
// References to fields with different but convertible type are allowed. For
// example, a string can receive its value from an int.
//
// A resource may declare an expression that is a combination of string
// literals and references as an input to a string field. This allows
// concatenating strings that will dynamically be resolved on runtime, based on
// outputs from parent resources.
func DecodeBody(body hcl.Body, ctx *DecodeContext, target *graph.Graph) (*config.Project, hcl.Diagnostics) {
	cont, diags := body.Content(rootSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	dec := &decoder{
		graph:  target,
		fields: make(map[graph.Field]field),
		names:  make(map[string]*hcl.Range),
	}

	var project *config.Project
	for _, b := range cont.Blocks {
		b := b // for setting label pointer
		switch b.Type {
		case "project":
			if req := requireLabels(b, "project name"); req.HasErrors() {
				diags = append(diags, req...)
				continue
			}
			project = &config.Project{}
			diags = append(diags, gohcl.DecodeBody(b.Body, nil, project)...)
			project.Name = b.Labels[0]
		case "resource":
			if req := requireLabels(b, "resource name"); req.HasErrors() {
				diags = append(diags, req...)
				continue
			}
			diags = append(diags, dec.decodeResource(b, ctx)...)
		}
	}

	diags = append(diags, dec.resolveValues()...)

	return project, diags
}

func requireLabels(block *hcl.Block, names ...string) hcl.Diagnostics {
	for i, name := range names {
		title := strings.ToUpper(name[:1]) + name[1:]
		label := block.Labels[i]
		if label == "" {
			return hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("%s not set", title),
					Detail:   fmt.Sprintf("A %s cannot be blank.", name),
					Subject:  block.LabelRanges[i].Ptr(),
					Context:  block.DefRange.Ptr(),
				},
			}
		}
	}
	return nil
}
