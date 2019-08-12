package aws

import (
	"context"
	"encoding/json"

	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

// DefaultPolicyVersion is set on a policy when the version is omitted.
const DefaultPolicyVersion = "2012-10-17"

// IAMPolicyDocument generates an IAM policy.
type IAMPolicyDocument struct {
	// Specify the version of the policy language that you want to use.
	// If not set, `2012-10-17` is used.
	Version *string `func:"input"`

	// Use this main policy element as a container for the following elements.
	// You can include more than one statement in a policy.
	Statements []IAMPolicyStatement `func:"input" name:"statement"`

	// Outputs

	JSON string `func:"output"`
}

// IAMPolicyStatement is a single statement in an IAM Policy Document.
type IAMPolicyStatement struct {
	// Include an optional statement ID to differentiate between your statements.
	ID *string

	// Use `Allow` or `Deny` to indicate whether the policy allows or
	// denies access.
	Effect string `validate:"oneof=Allow Deny"`

	// The account, user, role, or federated user to which you would like to
	// allow or deny access.
	//
	// If you are creating a policy to attach to a user or role, you cannot
	// include this element. The principal is implied as that user or role.
	Principals *map[string][]string

	// The account, user, role or federated user to which the statement does
	// **not** apply to.
	NotPrincipals *map[string][]string

	// Include a list of actions that the policy allows or denies.
	Actions *[]string

	// List of actions that the statement do **not** apply to.
	NotActions *[]string

	// List of resources to which the actions apply.
	Resources *[]string

	// List of resources to which the actions do **not** apply.
	NotResources *[]string

	// Specify the circumstances under which the policy grants permission.
	//
	//   condition = {
	//     "StringEquals" = {
	//       "aws:username": "johndoe"
	//     }
	//   }
	//
	// See https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html
	// for supported operators.
	Conditions *map[string]map[string]string
}

// Create creates a new IAM role.
func (p *IAMPolicyDocument) Create(ctx context.Context, r *resource.CreateRequest) error {
	return p.generate()
}

// Delete deletes the IAM role.
func (p *IAMPolicyDocument) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	return nil
}

// Update updates the IAM role.
func (p *IAMPolicyDocument) Update(ctx context.Context, r *resource.UpdateRequest) error {
	return p.generate()
}

type awsIAMPolicyDoc struct {
	Version    string             `json:"Version"`
	Statements []awsIAMPolicyStmt `json:"Statement"`
}

type awsIAMPolicyStmt struct {
	Sid          string                       `json:"Sid,omitempty"`
	Effect       string                       `json:"Effect"`                 // Allow / Deny
	Action       interface{}                  `json:"Action,omitempty"`       // string or []string
	NotAction    interface{}                  `json:"NotAction,omitempty"`    // string or []string
	Principal    map[string]interface{}       `json:"Principal,omitempty"`    // map to string or []string
	NotPrincipal map[string]interface{}       `json:"NotPrincipal,omitempty"` // map to string or []string
	Resource     interface{}                  `json:"Resource,omitempty"`     // string or []string
	NotResource  interface{}                  `json:"NotResource,omitempty"`  // string or []string
	Condition    map[string]map[string]string `json:"Condition,omitempty"`
}

func (p *IAMPolicyDocument) generate() error {
	doc := awsIAMPolicyDoc{}

	if p.Version != nil {
		doc.Version = *p.Version
	} else {
		doc.Version = DefaultPolicyVersion
	}

	for _, stmt := range p.Statements {
		s := awsIAMPolicyStmt{
			Effect: stmt.Effect,
		}
		if stmt.ID != nil {
			s.Sid = *stmt.ID
		}
		if stmt.Actions != nil {
			s.Action = stringOrSlice(*stmt.Actions)
		}
		if stmt.Resources != nil {
			s.Resource = stringOrSlice(*stmt.Resources)
		}
		if stmt.NotActions != nil {
			s.NotAction = stringOrSlice(*stmt.NotActions)
		}
		if stmt.NotResources != nil {
			s.NotResource = stringOrSlice(*stmt.NotResources)
		}
		if stmt.Principals != nil {
			s.Principal = make(map[string]interface{})
			for k, pp := range *stmt.Principals {
				s.Principal[k] = stringOrSlice(pp)
			}
		}
		if stmt.NotPrincipals != nil {
			s.NotPrincipal = make(map[string]interface{})
			for k, pp := range *stmt.NotPrincipals {
				s.NotPrincipal[k] = stringOrSlice(pp)
			}
		}
		if stmt.Conditions != nil {
			s.Condition = *stmt.Conditions
		}

		doc.Statements = append(doc.Statements, s)
	}

	j, err := json.Marshal(doc)
	if err != nil {
		return backoff.Permanent(err)
	}

	p.JSON = string(j)

	return nil
}

// stringOrSlice returns the first string only if the length is 1. Otherwise
// returns the original string slice.
func stringOrSlice(ss []string) interface{} {
	switch len(ss) {
	case 0:
		return nil
	case 1:
		return ss[0]
	default:
		return ss
	}
}
