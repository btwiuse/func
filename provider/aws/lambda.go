//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct LambdaFunction --template ../../tools/structdoc/template.txt --data type=aws_lambda_function --output ../../docs/resources/aws/lambda_function.md

package aws

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/lambdaiface"
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
		TargetArn *string `input:"target_arn"`
	} `input:"dead_letter_config"`

	// A description of the function.
	Description *string `input:"description"`

	// Environment variables that are accessible from function code during execution.
	Environment *struct {
		Variables map[string]string `input:"variables"`
	} `input:"environment"`

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
	FunctionName string `input:"function_name"`

	// The name of the method within your code that Lambda calls to execute
	// your function. For more information, see
	// [Programming Model](http://docs.aws.amazon.com/lambda/latest/dg/programming-model-v2.html).
	Handler string `input:"handler"`

	// The ARN of the KMS key used to encrypt your function's environment
	// variables. If not provided, AWS Lambda will use a default service key.
	KMSKeyArn *string `input:"kms_key_arn"`

	// A list of [function layers](http://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html)
	// to add to the function's execution environment.
	Layers *[]string `input:"layers"`

	// The amount of memory that your function has access to. Increasing the
	// function's memory also increases it's CPU allocation. The default value
	// is 128 MB. The value must be a multiple of 64 MB.
	MemorySize *int64 `input:"memory_size"`

	// Set to true to publish the first version of the function during
	// creation.
	Publish *bool `input:"publish"`

	// Region to run the Lambda function in.
	Region string `input:"region"`

	// The Amazon Resource Name (ARN) of the function's execution role
	// (http://docs.aws.amazon.com/lambda/latest/dg/intro-permission-model.html#lambda-intro-execution-role).
	Role string `input:"role"`

	// The runtime version for the function.
	//
	// Allowed values:
	//   - nodejs
	//   - nodejs4.3
	//   - nodejs6.10
	//   - nodejs8.10
	//   - java8
	//   - python2.7
	//   - python3.6
	//   - python3.7
	//   - dotnetcore1.0
	//   - dotnetcore2.0
	//   - dotnetcore2.1
	//   - nodejs4.3-edge
	//   - go1.x
	//   - ruby2.5
	//   - provided
	Runtime string `input:"runtime"` // TODO: enum

	// The list of tags (key-value pairs) assigned to the new function. For
	// more information, see
	// [Tagging Lambda Functions](http://docs.aws.amazon.com/lambda/latest/dg/tagging.html)
	// in the AWS Lambda Developer Guide.
	Tags *map[string]string `input:"tags"`

	// The amount of time that Lambda allows a function to run before
	// terminating it. The default is 3 seconds. The maximum allowed value is
	// 900 seconds.
	Timeout *int64 `input:"timeout"`

	// Set Mode to Active to sample and trace a subset of incoming requests
	// with AWS X-Ray.
	//
	// https://docs.aws.amazon.com/goto/WebAPI/lambda-2015-03-31/TracingConfig
	TracingConfig *struct {
		// The tracing mode.
		Mode string `input:"mode"`
	} `input:"tracing_config"`

	// If your Lambda function accesses resources in a VPC, you provide this parameter
	// identifying the list of security group IDs and subnet IDs. These must belong
	// to the same VPC. You must provide at least one security group and one subnet
	// ID.
	VpcConfig *struct {
		// A list of VPC security groups IDs.
		SecurityGroupIDs []string `input:"security_group_ids"`
		// A list of VPC subnet IDs.
		SubnetIds []string `input:"subnet_ids"`
	} `input:"vpc_config"`

	// Outputs

	// The SHA256 hash of the function's deployment package.
	CodeSha256 string `output:"code_sha_256"`

	// The size of the function's deployment package in bytes.
	CodeSize int64 `output:"code_size"`

	// The function's Amazon Resource Name.
	FunctionArn string `output:"function_arn"`

	// The date and time that the function was last updated.
	LastModified time.Time `output:"last_modified"`

	// The ARN of the master function.
	MasterArn *string `output:"master_arn"`

	// Represents the latest updated revision of the function or alias.
	RevisionID string `output:"revision_id"`

	// The version of the Lambda function.
	Version string `output:"version"`
}

// Type returns the type name for an AWS Lambda function.
func (p *LambdaFunction) Type() string { return "aws_lambda_function" }

// Create creates an AWS lambda function.
func (p *LambdaFunction) Create(ctx context.Context, r *resource.CreateRequest) error {
	if len(r.Source) == 0 {
		return errors.New("no source code provided")
	}
	if len(r.Source) > 1 {
		return errors.New("only one source archive allowed")
	}

	svc, err := lambdaService(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	src, err := r.Source[0].Reader(ctx)
	if err != nil {
		return errors.Wrap(err, "get source reader")
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
		MemorySize:   p.MemorySize,
		Publish:      p.Publish,
		Role:         aws.String(p.Role),
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
	if p.Layers != nil {
		input.Layers = *p.Layers
	}
	if p.Tags != nil {
		input.Tags = *p.Tags
	}
	if p.TracingConfig != nil {
		input.TracingConfig = &lambda.TracingConfig{
			Mode: lambda.TracingMode(p.TracingConfig.Mode),
		}
	}
	if p.VpcConfig != nil {
		input.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: p.VpcConfig.SecurityGroupIDs,
			SubnetIds:        p.VpcConfig.SubnetIds,
		}
	}

	req := svc.CreateFunctionRequest(input)
	req.SetContext(ctx)
	resp, err := req.Send()
	if err != nil {
		return err
	}

	// OK

	p.setFromResp(resp)

	return nil
}

// Delete deletes the lambda function.
func (p *LambdaFunction) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := lambdaService(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteFunctionRequest(&lambda.DeleteFunctionInput{
		FunctionName: aws.String(p.FunctionArn),
	})
	req.SetContext(ctx)
	_, err = req.Send()
	return err
}

// Update updates the lambda function.
func (p *LambdaFunction) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := lambdaService(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}
	if r.SourceChanged {
		if err := p.updateCode(ctx, svc, r); err != nil {
			return errors.Wrap(err, "update code")
		}
	}
	if r.ConfigChanged {
		if err := p.updateConfig(ctx, svc); err != nil {
			return errors.Wrap(err, "update config")
		}
	}
	return nil
}

func (p *LambdaFunction) updateCode(ctx context.Context, svc lambdaiface.LambdaAPI, r *resource.UpdateRequest) error {
	if len(r.Source) == 0 {
		return errors.New("no source code provided")
	}
	if len(r.Source) > 1 {
		return errors.New("only one source archive allowed")
	}

	src, err := r.Source[0].Reader(ctx)
	if err != nil {
		return errors.Wrap(err, "get source reader")
	}
	var zip bytes.Buffer
	if err := convert.Zip(&zip, src); err != nil {
		return errors.Wrap(err, "convert zip")
	}
	if err := src.Close(); err != nil {
		return errors.Wrap(err, "close source code")
	}

	req := svc.UpdateFunctionCodeRequest(&lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(p.FunctionName),
		ZipFile:      zip.Bytes(),
	})
	req.SetContext(ctx)
	resp, err := req.Send()
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	p.setFromResp(resp)

	return nil
}

func (p *LambdaFunction) updateConfig(ctx context.Context, svc lambdaiface.LambdaAPI) error {
	input := &lambda.UpdateFunctionConfigurationInput{
		Description:  p.Description,
		FunctionName: aws.String(p.FunctionName),
		Handler:      aws.String(p.Handler),
		KMSKeyArn:    p.KMSKeyArn,
		MemorySize:   p.MemorySize,
		Role:         aws.String(p.Role),
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
	if p.Layers != nil {
		input.Layers = *p.Layers
	}
	if p.TracingConfig != nil {
		input.TracingConfig = &lambda.TracingConfig{
			Mode: lambda.TracingMode(p.TracingConfig.Mode),
		}
	}
	if p.VpcConfig != nil {
		input.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: p.VpcConfig.SecurityGroupIDs,
			SubnetIds:        p.VpcConfig.SubnetIds,
		}
	}

	req := svc.UpdateFunctionConfigurationRequest(input)
	req.SetContext(ctx)
	resp, err := req.Send()
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	p.setFromResp(resp)

	return nil
}

func (p *LambdaFunction) setFromResp(resp *lambda.UpdateFunctionConfigurationOutput) {
	p.CodeSha256 = *resp.CodeSha256
	p.CodeSize = *resp.CodeSize
	p.FunctionArn = *resp.FunctionArn
	t, err := time.Parse(iso8601, *resp.LastModified)
	if err != nil {
		log.Printf("Could not parse Lambda modified timestamp %q, falling back to current time", *resp.LastModified)
		t = time.Now()
	}
	p.LastModified = t
	p.MasterArn = resp.MasterArn
	p.RevisionID = *resp.RevisionId
	p.Version = *resp.Version
}
