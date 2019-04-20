//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct LambdaInvokePermission --template ../../tools/structdoc/template.txt --data type=aws_lambda_permission --output ../../docs/resources/aws/lambda_permission.md

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/lambdaiface"
	"github.com/func/func/provider/aws/internal/config"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// LambdaInvokePermission sets permissions on a Lambda function.
type LambdaInvokePermission struct {
	// Inputs

	// The region the function is in.
	Region string `input:"region"`

	// The AWS Lambda action you want to allow in this statement. Each Lambda action
	// is a string starting with lambda: followed by the API name . For example,
	// lambda:CreateFunction. You can use wildcard (lambda:*) to grant permission
	// for all AWS Lambda actions.
	Action string `input:"action"`

	// A unique token that must be supplied by the principal invoking the function.
	// This is currently only used for Alexa Smart Home functions.
	EventSourceToken *string `input:"event_source_token"`

	// The name of the Lambda function.
	//
	// Name formats
	//
	//  - Function name - `MyFunction`.
	//  - Function ARN - `arn:aws:lambda:us-west-2:123456789012:function:MyFunction`.
	//  - Partial ARN - `123456789012:function:MyFunction`.
	//
	// The length constraint applies only to the full ARN. If you specify only
	// the function name, it is limited to 64 characters in length.
	FunctionName string `input:"function_name"`

	// The principal who is getting this permission. The principal can be an
	// AWS service (e.g. `s3.amazonaws.com` or `sns.amazonaws.com`) for service
	// triggers, or an account ID for cross-account access. If you specify a
	// service as a principal, use the SourceArn parameter to limit who can
	// invoke the function through that service.
	Principal string `input:"principal"`

	// Specify a version or alias to add permissions to a published version of the
	// function.
	Qualifier *string `input:"qualifier"`

	// An optional value you can use to ensure you are updating the latest update
	// of the function version or alias. If the RevisionID you pass doesn't match
	// the latest RevisionID of the function or alias, it will fail with an error
	// message.
	RevisionID *string `input:"revision_id"`

	// This parameter is used for S3 and SES. The AWS account ID (without a hyphen)
	// of the source owner. For example, if the SourceArn identifies a bucket, then
	// this is the bucket owner's account ID. You can use this additional condition
	// to ensure the bucket you specify is owned by a specific account (it is possible
	// the bucket owner deleted the bucket and some other AWS account created the
	// bucket). You can also use this condition to specify all sources (that is,
	// you don't specify the SourceArn) owned by a specific account.
	SourceAccount *string `input:"service_account"`

	// The Amazon Resource Name of the invoker.
	//
	// If you add a permission to a service principal without providing the source
	// ARN, any AWS account that creates a mapping to your function ARN can invoke
	// your Lambda function.
	SourceARN *string `input:"source_arn"`

	// A unique statement identifier.
	StatementID string `input:"statement_id"`

	// Outputs

	// The permission statement you specified in the request. The response returns
	// the same as a string using a backslash ("\") as an escape character in the
	// JSON.
	Statement string `output:"statement"`

	svc lambdaiface.LambdaAPI
}

// Type returns the type name for an AWS Lambda function.
func (l *LambdaInvokePermission) Type() string { return "aws_lambda_invoke_permission" }

// Create creates an AWS lambda function.
func (l *LambdaInvokePermission) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := l.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &lambda.AddPermissionInput{
		Action:           aws.String(l.Action),
		EventSourceToken: l.EventSourceToken,
		FunctionName:     aws.String(l.FunctionName),
		Principal:        aws.String(l.Principal),
		Qualifier:        l.Qualifier,
		RevisionId:       l.RevisionID,
		SourceAccount:    l.SourceAccount,
		SourceArn:        l.SourceARN,
		StatementId:      aws.String(l.StatementID),
	}

	req := svc.AddPermissionRequest(input)
	req.SetContext(ctx)
	resp, err := req.Send()
	if err != nil {
		return err
	}

	l.Statement = *resp.Statement

	return nil
}

// Delete deletes the lambda function.
func (l *LambdaInvokePermission) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := l.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.RemovePermissionRequest(&lambda.RemovePermissionInput{
		FunctionName: aws.String(l.FunctionName),
		Qualifier:    l.Qualifier,
		RevisionId:   l.RevisionID,
		StatementId:  aws.String(l.StatementID),
	})
	req.SetContext(ctx)
	_, err = req.Send()
	return err
}

// Update updates the lambda function.
func (l *LambdaInvokePermission) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Permission cannot be updated, delete and create nwe

	prev := r.Previous.(*LambdaInvokePermission)

	if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
		return errors.Wrap(err, "update-delete")
	}
	if err := l.Create(ctx, r.CreateRequest()); err != nil {
		return errors.Wrap(err, "update-create")
	}

	return nil
}

func (l *LambdaInvokePermission) service(auth resource.AuthProvider) (lambdaiface.LambdaAPI, error) {
	if l.svc == nil {
		cfg, err := config.WithRegion(auth, l.Region)
		if err != nil {
			return nil, errors.Wrap(err, "get aws config")
		}
		l.svc = lambda.New(cfg)
	}
	return l.svc, nil
}