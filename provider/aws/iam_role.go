package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/iamiface"
	"github.com/func/func/provider/aws/internal/config"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// IAMRole creates a new role for your AWS account. For more information about
// roles, go to [IAM Roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/WorkingWithRoles.html).
//
// For information about limitations on role names and the number of roles you
// can create, go to Limitations on
// [IAM Entities](http://docs.aws.amazon.com/IAM/latest/UserGuide/LimitationsOnEntities.html)
// in the IAM User Guide.
type IAMRole struct {
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
	AssumeRolePolicyDocument string `input:"assume_role_policy_document"`

	// A description of the role.
	Description *string `input:"description"`

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
	MaxSessionDuration *int64 `input:"max_session_duration"`

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
	Path *string `input:"path"`

	// The ARN of the policy that is used to set the permissions boundary for
	// the role.
	PermissionsBoundary *string `input:"permission_boundary"`

	// The name of the role to create.
	//
	// This parameter allows (per its
	// [regex pattern](http://wikipedia.org/wiki/regex) a string of characters
	// consisting of upper and lowercase alphanumeric characters with no
	// spaces. You can also include any of the following characters: _+=,.@-
	//
	// Role names are not distinguished by case. For example, you cannot create
	// roles named both "PRODROLE" and "prodrole".
	RoleName string `input:"role_name"`

	// The Amazon Resource Name (ARN) specifying the role.
	//
	// For more information about ARNs and how to use them in policies, see
	// [IAM Identifiers](http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the IAM User Guide guide.
	ARN string `output:"arn"`

	CreateDate time.Time `output:"create_date"`
	RoleID     string    `output:"role_id"`

	svc iamiface.IAMAPI
}

// Type returns the type name for an AWS IAM role.
func (i *IAMRole) Type() string { return "aws_iam_role" }

// Create creates a new IAM role.
func (i *IAMRole) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := i.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.CreateRoleRequest(&iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(i.AssumeRolePolicyDocument),
		Description:              i.Description,
		MaxSessionDuration:       i.MaxSessionDuration,
		Path:                     i.Path,
		PermissionsBoundary:      i.PermissionsBoundary,
		RoleName:                 aws.String(i.RoleName),
	})
	req.SetContext(ctx)
	res, err := req.Send()
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	i.ARN = *res.Role.Arn
	i.CreateDate = *res.Role.CreateDate
	i.RoleID = *res.Role.RoleId

	return nil
}

// Delete deletes the IAM role.
func (i *IAMRole) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := i.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteRoleRequest(&iam.DeleteRoleInput{
		RoleName: aws.String(i.RoleName),
	})
	req.SetContext(ctx)
	if _, err := req.Send(); err != nil {
		return errors.Wrap(err, "send request")
	}

	return nil
}

// Update updates the IAM role.
func (i *IAMRole) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := i.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.UpdateRoleRequest(&iam.UpdateRoleInput{
		RoleName:           aws.String(i.RoleName),
		Description:        i.Description,
		MaxSessionDuration: i.MaxSessionDuration,
	})
	req.SetContext(ctx)
	if _, err := req.Send(); err != nil {
		return errors.Wrap(err, "send request")
	}

	return nil
}

func (i *IAMRole) service(auth resource.AuthProvider) (iamiface.IAMAPI, error) {
	if i.svc == nil {
		cfg, err := config.DefaultRegion(auth)
		if err != nil {
			return nil, errors.Wrap(err, "get aws config")
		}
		i.svc = iam.New(cfg)
	}
	return i.svc, nil
}
