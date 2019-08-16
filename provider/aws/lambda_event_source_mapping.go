package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

// LambdaEventSourceMapping is a mapping from an event source to a lambda
// function.
type LambdaEventSourceMapping struct {
	// Inputs

	// The maximum number of items to retrieve in a single batch.
	//
	//    * Amazon Kinesis - Default 100. Max 10,000.
	//
	//    * Amazon DynamoDB Streams - Default 100. Max 1,000.
	//
	//    * Amazon Simple Queue Service - Default 10. Max 10.
	BatchSize *int64 `func:"input" validate:"min=1,max=10000"`

	// Disables the event source mapping to pause polling and invocation.
	Enabled *bool `func:"input"`

	// The Amazon Resource Name (ARN) of the event source.
	//
	//    * Amazon Kinesis - The ARN of the data stream or a stream consumer.
	//
	//    * Amazon DynamoDB Streams - The ARN of the stream.
	//
	//    * Amazon Simple Queue Service - The ARN of the queue.
	EventSourceARN string `func:"input" name:"event_source_arn" validate:"aws_arn"`

	// The name of the Lambda function.
	//
	// Name formats
	//
	//    * Function name - MyFunction.
	//
	//    * Function ARN - arn:aws:lambda:us-west-2:123456789012:function:MyFunction.
	//
	//    * Version or Alias ARN - arn:aws:lambda:us-west-2:123456789012:function:MyFunction:PROD.
	//
	//    * Partial ARN - 123456789012:function:MyFunction.
	//
	// The length constraint applies only to the full ARN. If you specify only the
	// function name, it's limited to 64 characters in length.
	FunctionName string `func:"input" validate:"min=1"`

	// The position in a stream from which to start reading. Required for Amazon
	// Kinesis and Amazon DynamoDB Streams sources. AT_TIMESTAMP is only supported
	// for Amazon Kinesis streams.
	StartingPosition *string `func:"input" validate:"oneof=TRIM_HORIZON LATEST AT_TIMESTAMP"`

	// With StartingPosition set to AT_TIMESTAMP, the RFC3339 formatted time from which to start reading.
	StartingPositionTimestamp *string `func:"input"`

	Region string `func:"input"`

	// Outputs

	// The ARN of the Lambda function.
	FunctionARN string `func:"output"`

	// RFC3339 formatted date for when the event source mapping was last updated.
	LastModified string `func:"output"`

	// The identifier of the event source mapping.
	UUID string `func:"output"`

	lambdaService
}

// Create creates an AWS lambda function.
func (p *LambdaEventSourceMapping) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &lambda.CreateEventSourceMappingInput{
		BatchSize:      p.BatchSize,
		Enabled:        p.Enabled,
		EventSourceArn: aws.String(p.EventSourceARN),
		FunctionName:   aws.String(p.FunctionName),
	}
	if p.StartingPosition != nil {
		input.StartingPosition = lambda.EventSourcePosition(*p.StartingPosition)
	}
	if p.StartingPositionTimestamp != nil {
		t, err := time.Parse(time.RFC3339, *p.StartingPositionTimestamp)
		if err != nil {
			return backoff.Permanent(err)
		}
		input.StartingPositionTimestamp = &t
	}

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateEventSourceMappingRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.FunctionARN = *resp.FunctionArn
	p.LastModified = resp.LastModified.Format(time.RFC3339)
	p.UUID = *resp.UUID

	return nil
}

// Delete deletes the lambda function.
func (p *LambdaEventSourceMapping) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &lambda.DeleteEventSourceMappingInput{
		UUID: aws.String(p.UUID),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteEventSourceMappingRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the lambda function.
func (p *LambdaEventSourceMapping) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	prev := r.Previous.(*LambdaEventSourceMapping)

	input := &lambda.UpdateEventSourceMappingInput{
		BatchSize:    p.BatchSize,
		Enabled:      p.Enabled,
		FunctionName: aws.String(p.FunctionName),
		UUID:         aws.String(prev.UUID),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.UpdateEventSourceMappingRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.FunctionARN = *resp.FunctionArn
	p.LastModified = resp.LastModified.Format(time.RFC3339)

	return nil
}
