package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

// IAMRole creates a new role for your AWS account. For more information about
// roles, go to [IAM Roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/WorkingWithRoles.html).
//
// For information about limitations on role names and the number of roles you
// can create, go to Limitations on
// [IAM Entities](http://docs.aws.amazon.com/IAM/latest/UserGuide/LimitationsOnEntities.html)
// in the IAM User Guide.
type IAMRole struct {
	// Inputs

	// The trust relationship policy document that grants an entity permission to
	// assume the role.
	//
	// The [regex pattern](http://wikipedia.org/wiki/regex) used to validate this
	// parameter is a string of characters consisting of the following:
	//
	// * Any printable ASCII character ranging from the space character (\u0020)
	//   through the end of the ASCII character range
	//
	// * The printable characters in the Basic Latin and Latin-1 Supplement character
	//   set (through \u00FF)
	//
	// * The special characters tab (\u0009), line feed (\u000A), and carriage
	//   return (\u000D)
	AssumeRolePolicyDocument string `func:"input"`

	// A description of the role.
	Description *string `func:"input"`

	// The maximum session duration (in seconds) that you want to set for the
	// specified role. If you do not specify a value for this setting, the
	// default maximum of one hour is applied. This setting can have a value
	// from 1 hour to 12 hours.
	//
	// Anyone who assumes the role from the AWS CLI or API can use the
	// DurationSeconds API parameter or the duration-seconds CLI parameter to
	// request a longer session. The MaxSessionDuration setting determines the
	// maximum duration that can be requested using the DurationSeconds
	// parameter. If users don't specify a value for the DurationSeconds
	// parameter, their security credentials are valid for one hour by default.
	// This applies when you use the AssumeRole* API operations or the
	// assume-role* CLI operations but does not apply when you use those
	// operations to create a console URL. For more information, see
	// [Using IAM Roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html)
	// in the IAM User Guide.
	MaxSessionDuration *int64 `func:"input" validate:"min=3600,max=43200"`

	// The path to the role. For more information about paths, see
	// [IAM Identifiers](http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the IAM User Guide.
	//
	// This parameter is optional. If it is not included, it defaults to a
	// slash (/).
	//
	// This parameter allows (per its
	// [regex pattern](http://wikipedia.org/wiki/regex) a string of characters
	// consisting of either a forward slash (/) by itself or a string that must
	// begin and end with forward slashes. In addition, it can contain any
	// ASCII character from the ! (\u0021) through the DEL character (\u007F),
	// including most punctuation characters, digits, and upper and lowercased
	// letters.
	Path *string `func:"input"`

	// The ARN of the policy that is used to set the permissions boundary for
	// the role.
	PermissionsBoundary *string `func:"input"`

	// Region to use for IAM API calls.
	//
	// IAM is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string `func:"input"`

	// The name of the role to create.
	//
	// This parameter allows (per its
	// [regex pattern](http://wikipedia.org/wiki/regex) a string of characters
	// consisting of upper and lowercase alphanumeric characters with no
	// spaces. You can also include any of the following characters: _+=,.@-
	//
	// Role names are not distinguished by case. For example, you cannot create
	// roles named both "PRODROLE" and "prodrole".
	RoleName string `func:"input"`

	// The Amazon Resource Name (ARN) specifying the role.
	ARN *string `func:"output"`

	// RFC3339 formatted date and time for when the role was created.
	CreateDate string `func:"output"`

	// The stable and unique string identifying the role.
	RoleID *string `func:"output"`

	iamService
}

// Create creates a new IAM role.
func (p *IAMRole) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(p.AssumeRolePolicyDocument),
		Description:              p.Description,
		MaxSessionDuration:       p.MaxSessionDuration,
		Path:                     p.Path,
		PermissionsBoundary:      p.PermissionsBoundary,
		RoleName:                 aws.String(p.RoleName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateRoleRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.ARN = resp.Role.Arn
	p.CreateDate = resp.Role.CreateDate.Format(time.RFC3339)
	p.RoleID = resp.Role.RoleId

	return nil
}

// Delete deletes the IAM role.
func (p *IAMRole) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.DeleteRoleInput{
		RoleName: aws.String(p.RoleName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteRoleRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the IAM role.
func (p *IAMRole) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.UpdateRoleInput{
		RoleName:           aws.String(p.RoleName),
		Description:        p.Description,
		MaxSessionDuration: p.MaxSessionDuration,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.UpdateRoleRequest(input).Send(ctx)
	return handlePutError(err)
}
