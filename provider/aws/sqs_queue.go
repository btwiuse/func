package aws

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// SQSQueue manages an AWS SQS queue.
//
// Amazon Simple Queue Service (SQS) is a fully managed message queuing service
// that enables you to decouple and scale microservices, distributed systems,
// and serverless applications. SQS eliminates the complexity and overhead
// associated with managing and operating message oriented middleware, and
// empowers developers to focus on differentiating work. Using SQS, you can
// send, store, and receive messages between software components at any volume,
// without losing messages or requiring other services to be available. Get
// started with SQS in minutes using the AWS console, Command Line Interface or
// SDK of your choice, and three simple commands.
//
// SQS offers two types of message queues. Standard queues offer maximum
// throughput, best-effort ordering, and at-least-once delivery. SQS FIFO
// queues are designed to guarantee that messages are processed exactly once,
// in the exact order that they are sent.
type SQSQueue struct {
	// Inputs

	// Attributes:

	// The length of time, in seconds, for which the delivery of all messages
	// in the queue is delayed. Valid values: An integer from 0 to 900 seconds
	// (15 minutes). Default: 0.
	Delay *int `func:"input" validate:"min=0,max=900"`

	// The limit of how many bytes a message can contain before Amazon SQS
	// rejects it. Valid values: An integer from 1,024 bytes (1 KiB) to 262,144
	// bytes (256 KiB). Default: 262,144 (256 KiB).
	MaximumMessageSize *int `func:"input" validate:"min=1024,max=262144"`

	// The length of time, in seconds, for which Amazon SQS retains a message.
	// Valid values: An integer from 60 seconds (1 minute) to 1,209,600 seconds
	// (14 days). Default: 345,600 (4 days).
	MessageRetentionPeriod *int `func:"input" validate:"min=60,max=1209600"`

	// The queue's policy. A valid AWS policy. For more information about
	// policy structure, see Overview of [AWS IAM
	// Policies](https://docs.aws.amazon.com/IAM/latest/UserGuide/PoliciesOverview.html)
	// in the Amazon IAM User Guide.
	Policy *string `func:"input"`

	// The length of time, in seconds, for which a ReceiveMessage action waits
	// for a message to arrive. Valid values: An integer from 0 to 20
	// (seconds). Default: 0.
	ReceiveMessageWaitTime *int `func:"input" validate:"min=0,max=20"`

	// Parameters for the dead-letter queue functionality of the source queue.
	// For more information about the redrive policy and dead-letter queues,
	// see [Using Amazon SQS Dead-Letter
	// Queues](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html)
	// in the Amazon Simple Queue Service Developer Guide.
	RedrivePolicy *struct {
		// The number of times a message is delivered to the source queue
		// before being moved to the dead-letter queue. When the ReceiveCount
		// for a message exceeds the maxReceiveCount for a queue, Amazon SQS
		// moves the message to the dead-letter-queue.
		MaxReceiveCount int `validate:"min=1" json:"maxReceiveCount"`

		// The Amazon Resource Name (ARN) of the dead-letter queue to which
		// Amazon SQS moves messages after the value of max_receive_count is
		// exceeded.
		//
		// Note: The dead-letter queue of a FIFO queue must also be a FIFO
		// queue. Similarly, the dead-letter queue of a standard queue must
		// also be a standard queue.
		DeadLetterTargetARN string `validate:"aws_arn" json:"deadLetterTargetArn"`
	} `func:"input"`

	// The visibility timeout for the queue, in seconds.
	// Valid values: An integer from 0 to 43,200 (12 hours). Default: 30.
	// For more information about the visibility timeout, see [Visibility
	// Timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
	// in the Amazon Simple Queue Service Developer Guide.
	VisibilityTimeout *int `func:"input" validate:"min=0,max=43200"`

	// The ID of an AWS-managed customer master key (CMK) for Amazon SQS or a
	// custom CMK. For more information, see [Key
	// Terms](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-server-side-encryption.html).
	// While the alias of the AWS-managed CMK for Amazon SQS is always
	// alias/aws/sqs, the alias of a custom CMK can, for example, be
	// alias/MyAlias . For more examples, see
	// [KeyId](https://docs.aws.amazon.com/kms/latest/APIReference/API_DescribeKey.html#API_DescribeKey_RequestParameters)
	// in the AWS Key Management Service API Reference.
	KMSMasterKeyID *string `func:"input" name:"kms_master_key_id"`

	// The length of time, in seconds, for which Amazon SQS can reuse a [data
	// key](https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#data-keys)
	// to encrypt or decrypt messages before calling AWS KMS again.  An integer
	// representing seconds, between 60 seconds (1 minute) and 86,400 seconds
	// (24 hours). Default: 300 (5 minutes).
	//
	// A shorter time period provides better security but results in more calls
	// to KMS which might incur charges after Free Tier. For more information,
	// see [How Does the Data Key Reuse Period
	// Work?](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-server-side-encryption.html).
	KMSDataKeyReusePeriod *int `func:"input" name:"kms_data_key_reuse_period" validate:"min=60,max=86400"`

	// Designates a queue as FIFO. If you don't specify the FifoQueue
	// attribute, Amazon SQS creates a standard queue. You can provide this
	// attribute only during queue creation. You can't change it for an
	// existing queue. When you set this attribute, you must also provide the
	// MessageGroupId for your messages explicitly. For more information, see
	// [FIFO Queue
	// Logic](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues.html)
	// in the Amazon Simple Queue Service Developer Guide.
	FIFOQueue *bool `func:"input" name:"fifo_queue"`

	// Enables content-based deduplication. For more information, see
	// [Exactly-Once
	// Processing](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues.html)
	// in the Amazon Simple Queue Service Developer Guide.
	//
	// Every message must have a unique MessageDeduplicationId, You may provide
	// a MessageDeduplicationId explicitly. If you aren't able to provide a
	// MessageDeduplicationId and you enable ContentBasedDeduplication for your
	// queue, Amazon SQS uses a SHA-256 hash to generate the
	// MessageDeduplicationId using the body of the message (but not the
	// attributes of the message). If you don't provide a
	// MessageDeduplicationId and the queue doesn't have
	// ContentBasedDeduplication set, the action fails with an error. If the
	// queue has ContentBasedDeduplication set, your MessageDeduplicationId
	// overrides the generated one. When ContentBasedDeduplication is in
	// effect, messages with identical content sent within the deduplication
	// interval are treated as duplicates and only one copy of the message is
	// delivered. If you send one message with ContentBasedDeduplication
	// enabled and then another message with a MessageDeduplicationId that is
	// the same as the one generated for the first MessageDeduplicationId, the
	// two messages are treated as duplicates and only one copy of the message
	// is delivered.
	ContentBasedDeduplication *bool `func:"input"`

	// The name of the new queue. The following limits apply to this name:
	//
	//    * A queue name can have up to 80 characters.
	//
	//    * Valid values: alphanumeric characters, hyphens (-), and underscores
	//    (_).
	//
	//    * A FIFO queue name must end with the .fifo suffix.
	//
	// Queue URLs and names are case-sensitive.
	//
	// QueueName is a required field
	QueueName string `func:"input"`

	// The region to create the queue in.
	Region string `func:"input"`

	// Outputs

	QueueURL string `func:"output"`
	QueueARN string `func:"output"`

	sqsService
}

// Create creates a new rest api.
func (p *SQSQueue) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &sqs.CreateQueueInput{
		QueueName:  aws.String(p.QueueName),
		Attributes: p.attributes(p),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateQueueRequest(input).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok {
			if aerr.Code() == sqs.ErrCodeQueueDeletedRecently {
				// Retry
				return err
			}
		}
		return handlePutError(err)
	}

	p.QueueURL = *resp.CreateQueueOutput.QueueUrl

	arn := p.QueueURL

	// ARN is not returned but it is very useful to have it immediately.
	// We can safely convert the url to ARN.
	arn = strings.Replace(arn, "https://sqs.", "arn:aws:sqs:", 1)
	arn = strings.Replace(arn, ".amazonaws.com/", ":", 1)
	arn = strings.Replace(arn, "/"+p.QueueName, ":"+p.QueueName, 1)
	p.QueueARN = arn

	return nil
}

// Delete removes an SQS queue.
func (p *SQSQueue) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &sqs.DeleteQueueInput{
		QueueUrl: aws.String(p.QueueURL),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}
	_, err = svc.DeleteQueueRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the attributes of an SQS queue.
//
// For now, only updating attributes is supported.
func (p *SQSQueue) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	prev := r.Previous.(*SQSQueue)

	input := &sqs.SetQueueAttributesInput{
		Attributes: p.attributes(p),
		QueueUrl:   aws.String(prev.QueueURL),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}
	_, err = svc.SetQueueAttributesRequest(input).Send(ctx)
	return handlePutError(err)
}

func (SQSQueue) attributes(p *SQSQueue) map[string]string {
	attr := make(map[string]string)
	if p.Delay != nil {
		attr["DelaySeconds"] = strconv.Itoa(*p.Delay)
	}
	if p.MaximumMessageSize != nil {
		attr["MaximumMessageSize"] = strconv.Itoa(*p.MaximumMessageSize)
	}
	if p.MessageRetentionPeriod != nil {
		attr["MessageRetentionPeriod"] = strconv.Itoa(*p.MessageRetentionPeriod)
	}
	if p.Policy != nil {
		attr["MessageRetentionPeriod"] = *p.Policy
	}
	if p.ReceiveMessageWaitTime != nil {
		attr["ReceiveMessageWaitTimeSeconds"] = strconv.Itoa(*p.ReceiveMessageWaitTime)
	}
	if p.RedrivePolicy != nil {
		j, _ := json.Marshal(p.RedrivePolicy)
		attr["RedrivePolicy"] = string(j)
	}
	if p.VisibilityTimeout != nil {
		attr["VisibilityTimeout"] = strconv.Itoa(*p.VisibilityTimeout)
	}
	if p.KMSMasterKeyID != nil {
		attr["KmsMasterKeyId"] = *p.KMSMasterKeyID
	}
	if p.KMSDataKeyReusePeriod != nil {
		attr["KmsDataKeyReusePeriodSeconds"] = strconv.Itoa(*p.KMSDataKeyReusePeriod)
	}

	// NOTE(akupila): These don't work now for some reason.
	if p.FIFOQueue != nil && *p.FIFOQueue {
		attr["FifoQueue"] = "true"
	}
	if p.ContentBasedDeduplication != nil && *p.ContentBasedDeduplication {
		attr["ContentBasedDeduplication"] = "true"
	}

	if len(attr) == 0 {
		// Set at least one attribute, otherwise the following error is returned:
		//   MalformedInput: End of list found where not expected
		attr["DelaySeconds"] = "0"
	}
	return attr
}
