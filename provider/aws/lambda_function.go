package aws

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/lambdaiface"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
	"github.com/func/func/source/convert"
	"github.com/pkg/errors"
)

// iso8601 is almost similar to iso3339, except the timezone is specified as
// +0000 instead of +00:00.
const iso8601 = "2006-01-02T15:04:05.999-0700"

// LambdaFunction manages AWS Lambda Functions.
//
// AWS Lambda lets you run code without provisioning or managing servers. You
// pay only for the compute time you consume - there is no charge when your
// code is not running.
//
// With Lambda, you can run code for virtually any type of application or
// backend service - all with zero administration. Just upload your code and
// Lambda takes care of everything required to run and scale your code with
// high availability. You can set up your code to automatically trigger from
// other AWS services or call it directly from any web or mobile app.
//
// https://aws.amazon.com/lambda/
type LambdaFunction struct {
	// Inputs

	// A dead letter queue configuration that specifies the queue or topic
	// where Lambda sends asynchronous events when they fail processing. For
	// more information, see
	// [Dead Letter Queues](http://docs.aws.amazon.com/lambda/latest/dg/dlq.html).
	DeadLetterConfig *struct {
		TargetArn *string `validate:"aws_arn"`
	} `func:"input"`

	// A description of the function.
	Description *string `func:"input"`

	// Environment variables that are accessible from function code during execution.
	Environment *struct {
		Variables map[string]string
	} `func:"input"`

	// The name of the Lambda function.
	//
	// Name formats
	//
	//   * Function name: `MyFunction`.
	//   * Function ARN:  `arn:aws:lambda:us-west-2:123456789012:function:MyFunction`.
	//   * Partial ARN:   `123456789012:function:MyFunction`.
	//
	// The length constraint applies only to the full ARN. If you specify only
	// the function name, it is limited to 64 characters in length.
	FunctionName string `func:"input" validation:"min=1,max=64"`

	// The name of the method within your code that Lambda calls to execute
	// your function. For more information, see
	// [Programming Model](http://docs.aws.amazon.com/lambda/latest/dg/programming-model-v2.html).
	Handler string `func:"input"`

	// The ARN of the KMS key used to encrypt your function's environment
	// variables. If not provided, AWS Lambda will use a default service key.
	KMSKeyArn *string `func:"input"`

	// A list of [function layers](http://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html)
	// to add to the function's execution environment.
	Layers []string `func:"input"`

	// The amount of memory that your function has access to. Increasing the
	// function's memory also increases it's CPU allocation. The default value
	// is 128 MB. The value must be a multiple of 64 MB.
	MemorySize *int64 `func:"input" validate:"min=64,max=3008,div=64"`

	// Set to true to publish the first version of the function during
	// creation.
	Publish *bool `func:"input"`

	// Region to run the Lambda function in.
	Region string `func:"input"`

	// The Amazon Resource Name (ARN) of the function's execution role
	// (http://docs.aws.amazon.com/lambda/latest/dg/intro-permission-model.html#lambda-intro-execution-role).
	Role string `func:"input" validate:"aws_arn"`

	// The runtime version for the function.
	Runtime string `func:"input" validate:"oneof=nodejs8.10 nodejs10.x java8 python2.7 python3.6 python3.7 dotnetcore1.0 dotnetcore2.0 dotnetcore2.1 go1.x ruby2.5 provided"` // nolint :lll

	// The list of tags (key-value pairs) assigned to the new function. For
	// more information, see
	// [Tagging Lambda Functions](http://docs.aws.amazon.com/lambda/latest/dg/tagging.html)
	// in the AWS Lambda Developer Guide.
	Tags map[string]string `func:"input"`

	// The amount of time that Lambda allows a function to run before
	// terminating it. The default is 3 seconds. The maximum allowed value is
	// 900 seconds.
	Timeout *int64 `func:"input" validate:"min=1,max=900"`

	// Set Mode to Active to sample and trace a subset of incoming requests
	// with AWS X-Ray.
	//
	// https://docs.aws.amazon.com/goto/WebAPI/lambda-2015-03-31/TracingConfig
	TracingConfig *struct {
		// The tracing mode.
		Mode string `validate:"oneof=Active PassTrough"`
	} `func:"input"`

	// If your Lambda function accesses resources in a VPC, you provide this parameter
	// identifying the list of security group IDs and subnet IDs. These must belong
	// to the same VPC. You must provide at least one security group and one subnet
	// ID.
	VPCConfig *struct {
		// A list of VPC security groups IDs.
		SecurityGroupIDs []string
		// A list of VPC subnet IDs.
		SubnetIDs []string
	} `func:"input" name:"vpc_config"`

	// Outputs

	// The SHA256 hash of the function's deployment package.
	CodeSha256 *string `func:"output"`

	// The size of the function's deployment package in bytes.
	CodeSize *int64 `func:"output"`

	// The function's Amazon Resource Name.
	FunctionARN *string `func:"output"`

	// The date and time that the function was last updated.
	LastModified time.Time `func:"output"`

	// The ARN of the master function.
	MasterARN *string `func:"output"`

	// Represents the latest updated revision of the function or alias.
	RevisionID *string `func:"output"`

	// The version of the Lambda function.
	Version *string `func:"output"`

	lambdaService
}

// Create creates an AWS lambda function.
func (p *LambdaFunction) Create(ctx context.Context, r *resource.CreateRequest) error {
	if len(r.Source) == 0 {
		return backoff.Permanent(fmt.Errorf("no source code provided"))
	}
	if len(r.Source) > 1 {
		return backoff.Permanent(fmt.Errorf("only one source archive allowed"))
	}

	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	src, err := r.Source[0].Reader(ctx)
	if err != nil {
		return backoff.Permanent(errors.Wrap(err, "get source reader"))
	}
	var zip bytes.Buffer
	if err := convert.Zip(&zip, src); err != nil {
		return errors.Wrap(err, "convert zip")
	}
	if err := src.Close(); err != nil {
		return errors.Wrap(err, "close source code")
	}

	input := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: zip.Bytes(),
		},
		Description:  p.Description,
		FunctionName: aws.String(p.FunctionName),
		Handler:      aws.String(p.Handler),
		KMSKeyArn:    p.KMSKeyArn,
		Layers:       p.Layers,
		MemorySize:   p.MemorySize,
		Publish:      p.Publish,
		Role:         aws.String(p.Role),
		Runtime:      lambda.Runtime(p.Runtime),
		Tags:         p.Tags,
		Timeout:      p.Timeout,
	}
	if p.DeadLetterConfig != nil {
		input.DeadLetterConfig = &lambda.DeadLetterConfig{
			TargetArn: p.DeadLetterConfig.TargetArn,
		}
	}
	if p.Environment != nil {
		input.Environment = &lambda.Environment{
			Variables: p.Environment.Variables,
		}
	}
	if p.TracingConfig != nil {
		input.TracingConfig = &lambda.TracingConfig{
			Mode: lambda.TracingMode(p.TracingConfig.Mode),
		}
	}
	if p.VPCConfig != nil {
		input.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: p.VPCConfig.SecurityGroupIDs,
			SubnetIds:        p.VPCConfig.SubnetIDs,
		}
	}

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateFunctionRequest(input).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == lambda.ErrCodeInvalidParameterValueException &&
				aerr.Message() == "The role defined for the function cannot be assumed by Lambda." {
				// This happens when the IAM role has not provisioned yet, as
				// IAM is eventually consistent. The same call will succeed
				// within ~10 seconds.
				return err
			}
		}
		return handlePutError(err)
	}

	// OK

	p.CodeSha256 = resp.CodeSha256
	p.CodeSize = resp.CodeSize
	p.FunctionARN = resp.FunctionArn
	t, err := time.Parse(iso8601, *resp.LastModified)
	if err != nil {
		log.Printf("Could not parse Lambda modified timestamp %q, falling back to current time", *resp.LastModified)
		t = time.Now()
	}
	p.LastModified = t
	p.MasterARN = resp.MasterArn
	p.RevisionID = resp.RevisionId
	p.Version = resp.Version

	return nil
}

// Delete deletes the lambda function.
func (p *LambdaFunction) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &lambda.DeleteFunctionInput{
		FunctionName: p.FunctionARN,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteFunctionRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the lambda function.
func (p *LambdaFunction) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}
	if r.SourceChanged {
		if err := p.updateCode(ctx, svc, r); err != nil {
			return err
		}
	}
	if r.ConfigChanged {
		if err := p.updateConfig(ctx, svc, r); err != nil {
			return err
		}
	}
	return nil
}

func (p *LambdaFunction) updateCode(ctx context.Context, svc lambdaiface.ClientAPI, r *resource.UpdateRequest) error {
	if len(r.Source) == 0 {
		return backoff.Permanent(fmt.Errorf("no source code provided"))
	}
	if len(r.Source) > 1 {
		return backoff.Permanent(fmt.Errorf("only one source archive allowed"))
	}

	prev := r.Previous.(*LambdaFunction)

	src, err := r.Source[0].Reader(ctx)
	if err != nil {
		return errors.Wrap(err, "get source reader")
	}
	var zip bytes.Buffer
	if err := convert.Zip(&zip, src); err != nil {
		return backoff.Permanent(errors.Wrap(err, "convert zip"))
	}
	if err := src.Close(); err != nil {
		return backoff.Permanent(errors.Wrap(err, "close source code"))
	}

	input := &lambda.UpdateFunctionCodeInput{
		FunctionName: prev.FunctionARN,
		ZipFile:      zip.Bytes(),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.UpdateFunctionCodeRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.CodeSha256 = resp.CodeSha256
	p.CodeSize = resp.CodeSize
	p.FunctionARN = resp.FunctionArn
	t, err := time.Parse(iso8601, *resp.LastModified)
	if err != nil {
		log.Printf("Could not parse Lambda modified timestamp %q, falling back to current time", *resp.LastModified)
		t = time.Now()
	}
	p.LastModified = t
	p.MasterARN = resp.MasterArn
	p.RevisionID = resp.RevisionId
	p.Version = resp.Version

	return nil
}

func (p *LambdaFunction) updateConfig(ctx context.Context, svc lambdaiface.ClientAPI, r *resource.UpdateRequest) error {
	prev := r.Previous.(*LambdaFunction)
	input := &lambda.UpdateFunctionConfigurationInput{
		Description:  p.Description,
		FunctionName: prev.FunctionARN,
		Handler:      aws.String(p.Handler),
		KMSKeyArn:    p.KMSKeyArn,
		MemorySize:   p.MemorySize,
		Role:         aws.String(p.Role),
		Layers:       p.Layers,
		Runtime:      lambda.Runtime(p.Runtime),
		Timeout:      p.Timeout,
	}
	if p.DeadLetterConfig != nil {
		input.DeadLetterConfig = &lambda.DeadLetterConfig{
			TargetArn: p.DeadLetterConfig.TargetArn,
		}
	}
	if p.Environment != nil {
		input.Environment = &lambda.Environment{
			Variables: p.Environment.Variables,
		}
	}
	if p.TracingConfig != nil {
		input.TracingConfig = &lambda.TracingConfig{
			Mode: lambda.TracingMode(p.TracingConfig.Mode),
		}
	}
	if p.VPCConfig != nil {
		input.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: p.VPCConfig.SecurityGroupIDs,
			SubnetIds:        p.VPCConfig.SubnetIDs,
		}
	}

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.UpdateFunctionConfigurationRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.CodeSha256 = resp.CodeSha256
	p.CodeSize = resp.CodeSize
	p.FunctionARN = resp.FunctionArn
	t, err := time.Parse(iso8601, *resp.LastModified)
	if err != nil {
		log.Printf("Could not parse Lambda modified timestamp %q, falling back to current time", *resp.LastModified)
		t = time.Now()
	}
	p.LastModified = t
	p.MasterARN = resp.MasterArn
	p.RevisionID = resp.RevisionId
	p.Version = resp.Version

	return nil
}
