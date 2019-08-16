package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
)

// DynamoDBTable provides a DynamoDB table.
//
// Amazon DynamoDB is a key-value and document database that delivers
// single-digit millisecond performance at any scale. It's a fully managed,
// multiregion, multimaster, durable database with built-in security, backup
// and restore, and in-memory caching for internet-scale applications. DynamoDB
// can handle more than 10 trillion requests per day and can support peaks of
// more than 20 million requests per second.
//
// Many of the world's fastest growing businesses such as Lyft, Airbnb, and
// Redfin as well as enterprises such as Samsung, Toyota, and Capital One
// depend on the scale and performance of DynamoDB to support their
// mission-critical workloads.
//
// Hundreds of thousands of AWS customers have chosen DynamoDB as their
// key-value and document database for mobile, web, gaming, ad tech, IoT, and
// other applications that need low-latency data access at any scale. Create a
// new table for your application and let DynamoDB handle the rest.
//
// https://aws.amazon.com/dynamodb/
type DynamoDBTable struct {
	// Inputs

	// An array of attributes that describe the key schema for the table and indexes.
	Attributes []struct {
		// A name for the attribute.
		Name string

		// The data type for the attribute, where:
		//
		//   * S - the attribute is of type String
		//   * N - the attribute is of type Number
		//   * B - the attribute is of type Binary
		Type string `validate:"oneof=S N B"`
	} `func:"input" validate:"min=1" name:"attribute"`

	// Controls how you are charged for read and write throughput and how you manage
	// capacity. This setting can be changed later.
	//
	//   * PROVISIONED - Sets the billing mode to PROVISIONED. We recommend using
	//   PROVISIONED for predictable workloads.
	//   * PAY_PER_REQUEST - Sets the billing mode to PAY_PER_REQUEST. We recommend
	//   using PAY_PER_REQUEST for unpredictable workloads.
	BillingMode string `func:"input" validate:"oneof=PROVISIONED PAY_PER_REQUEST"`

	// One or more global secondary indexes (the maximum is 20) to be created on
	// the table.
	GlobalSecondaryIndexes []struct {
		// The name of the global secondary index. The name must be unique
		// among all other indexes on this table.
		Name string `validate:"min=3"`

		// The complete key schema for a global secondary index
		KeySchema []struct {
			// The name of a key attribute.
			Name string `validate:"min=1"`

			// The role that this key attribute will assume:
			//
			//    * HASH - partition key
			//
			//    * RANGE - sort key
			//
			// The partition key of an item is also known as its hash
			// attribute. The term "hash attribute" derives from DynamoDB'
			// usage of an internal hash function to evenly distribute data
			// items across partitions, based on their partition key values.
			//
			// The sort key of an item is also known as its range attribute.
			// The term "range attribute" derives from the way DynamoDB stores
			// items with the same partition key physically close together, in
			// sorted order by the sort key value.
			Type string `validate:"oneof=HASH RANGE"`
		} `validate:"min=1"`

		// Represents attributes that are copied (projected) from the table
		// into the global secondary index. These are in addition to the
		// primary key attributes and index key attributes, which are
		// automatically projected.
		Projection struct {
			// Represents the non-key attribute names which will be projected
			// into the index.
			//
			// For local secondary indexes, the total count of NonKeyAttributes
			// summed across all of the local secondary indexes, must not
			// exceed 20. If you project the same attribute into two different
			// indexes, this counts as two distinct attributes when determining
			// the total.
			NonKeyAttributes []string `validate:"min=1"`

			// The set of attributes that are projected into the index:
			//
			//    * KEYS_ONLY - Only the index and primary keys are projected
			//    into the index.
			//
			//    * INCLUDE - Only the specified table attributes are projected
			//    into the index. The list of projected attributes are in
			//    NonKeyAttributes.
			//
			//    * ALL - All of the table attributes are projected into the
			//    index.
			Type string `validate:"oneof=KEYS_ONLY INCLUDE ALL"`
		}

		// Represents the provisioned throughput settings for a specified table
		// or index.  The settings can be modified using the UpdateTable
		// operation.
		//
		// For current minimum and maximum provisioned throughput values, see
		// [Limits](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Limits.html)
		// in the Amazon DynamoDB Developer Guide.  Please also see
		// https://docs.aws.amazon.com/goto/WebAPI/dynamodb-2012-08-10/ProvisionedThroughput
		ProvisionedThroughput *struct {
			// The maximum number of strongly consistent reads consumed per
			// second before DynamoDB returns a ThrottlingException.
			//
			// If read/write capacity mode is PAY_PER_REQUEST the value is set
			// to 0.
			ReadCapacityUnits int64

			// The maximum number of writes consumed per second before DynamoDB
			// returns a ThrottlingException.
			//
			// If read/write capacity mode is PAY_PER_REQUEST the value is set
			// to 0.
			WriteCapacityUnits int64
		}
	} `func:"input" validate:"max=20" name:"global_secondary_index"`

	// Specifies the attributes that make up the primary key for a table or an
	// index.  The attributes in KeySchema must also be defined in the
	// AttributeDefinitions array. For more information, see [Data
	// Model](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/DataModel.html)
	// in the Amazon DynamoDB Developer Guide.
	//
	// Each KeySchemaElement in the array is composed of:
	//
	//    * AttributeName - The name of this key attribute.
	//
	//    * KeyType - The role that the key attribute will assume: HASH -
	//    partition key RANGE - sort key
	//
	// For a simple primary key (partition key), you must provide exactly one
	// element with a KeyType of HASH.
	//
	// For a composite primary key (partition key and sort key), you must
	// provide exactly two elements, in this order: The first element must have
	// a KeyType of HASH, and the second element must have a KeyType of RANGE.
	//
	// For more information, see [Working with
	// Tables](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/WorkingWithTables.html)
	// in the Amazon DynamoDB Developer Guide.
	KeySchema []struct {
		// The name of a key attribute.
		Name string `validate:"min=1"`

		// The role that this key attribute will assume:
		//
		//    * HASH - partition key
		//
		//    * RANGE - sort key
		//
		// The partition key of an item is also known as its hash attribute. The term
		// "hash attribute" derives from DynamoDB' usage of an internal hash function
		// to evenly distribute data items across partitions, based on their partition
		// key values.
		//
		// The sort key of an item is also known as its range attribute. The term "range
		// attribute" derives from the way DynamoDB stores items with the same partition
		// key physically close together, in sorted order by the sort key value.
		Type string `validate:"oneof=HASH RANGE"`
	} `func:"input" validate:"min=1"`

	// One or more local secondary indexes (the maximum is 5) to be created on
	// the table. Each index is scoped to a given partition key value. There is
	// a 10 GB size limit per partition key value; otherwise, the size of a
	// local secondary index is unconstrained.
	//
	// Each local secondary index in the array includes the following:
	//
	//    * IndexName - The name of the local secondary index. Must be unique
	//    only for this table.
	//
	//    * KeySchema - Specifies the key schema for the local secondary index.
	//    The key schema must begin with the same partition key as the table.
	//
	//    * Projection - Specifies attributes that are copied (projected) from
	//    the table into the index. These are in addition to the primary key
	//    attributes and index key attributes, which are automatically
	//    projected. Each attribute specification is composed of:
	//    ProjectionType - One of the following: KEYS_ONLY - Only the index and
	//    primary keys are projected into the index. INCLUDE - Only the
	//    specified table attributes are projected into the index. The list of
	//    projected attributes is in NonKeyAttributes. ALL - All of the table
	//    attributes are projected into the index. NonKeyAttributes - A list of
	//    one or more non-key attribute names that are projected into the
	//    secondary index. The total count of attributes provided in
	//    NonKeyAttributes, summed across all of the secondary indexes, must
	//    not exceed 100. If you project the same attribute into two different
	//    indexes, this counts as two distinct attributes when determining the
	//    total.
	LocalSecondaryIndexes []struct {
		// The name of the local secondary index. The name must be unique among
		// all other indexes on this table.
		Name string `validate:"min=3"`

		// The complete key schema for the local secondary index.
		KeySchema []struct {
			// The name of a key attribute.
			Name string `validate:"min=1"`

			// The role that this key attribute will assume:
			//
			//    * HASH - partition key
			//
			//    * RANGE - sort key
			//
			// The partition key of an item is also known as its hash
			// attribute. The term "hash attribute" derives from DynamoDB'
			// usage of an internal hash function to evenly distribute data
			// items across partitions, based on their partition key values.
			//
			// The sort key of an item is also known as its range attribute.
			// The term "range attribute" derives from the way DynamoDB stores
			// items with the same partition key physically close together, in
			// sorted order by the sort key value.
			Type string `validate:"oneof=HASH RANGE"`
		} `validate:"min=1"`

		// Represents attributes that are copied (projected) from the table
		// into the local secondary index. These are in addition to the primary
		// key attributes and index key attributes, which are automatically
		// projected.
		Projection struct {
			// Represents the non-key attribute names which will be projected
			// into the index.
			//
			// For local secondary indexes, the total count of NonKeyAttributes
			// summed across all of the local secondary indexes, must not
			// exceed 20. If you project the same attribute into two different
			// indexes, this counts as two distinct attributes when determining
			// the total.
			NonKeyAttributes []string `validate:"min=1"`

			// The set of attributes that are projected into the index:
			//
			//    * KEYS_ONLY - Only the index and primary keys are projected
			//    into the index.
			//
			//    * INCLUDE - Only the specified table attributes are projected
			//    into the index. The list of projected attributes are in
			//    NonKeyAttributes.
			//
			//    * ALL - All of the table attributes are projected into the
			//    index.
			Type string `validate:"oneof=KEYS_ONLY INCLUDE ALL"`
		}
	} `func:"input" validate:"max=3" name:"local_secondary_index"`

	// Represents the provisioned throughput settings for a specified table or
	// index.  The settings can be modified using the UpdateTable operation.
	//
	// If you set BillingMode as PROVISIONED, you must specify this property.
	// If you set BillingMode as PAY_PER_REQUEST, you cannot specify this
	// property.
	//
	// For current minimum and maximum provisioned throughput values, see
	// [Limits](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Limits.html)
	// in the Amazon DynamoDB Developer Guide.
	ProvisionedThroughput *struct {
		// The maximum number of strongly consistent reads consumed per second
		// before DynamoDB returns a ThrottlingException.
		//
		// If read/write capacity mode is PAY_PER_REQUEST the value is set to
		// 0.
		ReadCapacityUnits int64

		// The maximum number of writes consumed per second before DynamoDB
		// returns a ThrottlingException.
		//
		// If read/write capacity mode is PAY_PER_REQUEST the value is set to
		// 0.
		WriteCapacityUnits int64
	} `func:"input"`

	// Represents the settings used to enable server-side encryption.
	SSE *struct {
		// Indicates whether server-side encryption is done using an AWS
		// managed CMK or an AWS owned CMK. If enabled (true), server-side
		// encryption type is set to KMS and an AWS managed CMK is used (AWS
		// KMS charges apply). If disabled (false) or not specified,
		// server-side encryption is set to AWS owned CMK.
		Enabled *bool

		// The KMS Customer Master Key (CMK) which should be used for the KMS
		// encryption.  To specify a CMK, use its key ID, Amazon Resource Name
		// (ARN), alias name, or alias ARN. Note that you should only provide
		// this parameter if the key is different from the default DynamoDB
		// Customer Master Key alias/aws/dynamodb.
		KMSMasterKeyID *string `name:"kms_master_key_id"`

		// Note(akupila): Type is disabled as only KMS is supported. The output
		// will contain KMS.
		//
		// Server-side encryption type. The only supported value is:
		//
		//    * KMS - Server-side encryption which uses AWS Key Management
		//    Service. Key is stored in your account and is managed by AWS KMS
		//    (KMS charges apply).
		// Type string
	} `func:"input"`

	// The settings for DynamoDB Streams on the table.
	Stream *struct {
		// Indicates whether DynamoDB Streams is enabled (true) or disabled
		// (false) on the table.
		Enabled *bool

		// When an item in the table is modified, ViewType determines what
		// information is written to the stream for this table. Valid values
		// for StreamViewType are:
		//
		//    * KEYS_ONLY - Only the key attributes of the modified item are written
		//    to the stream.
		//
		//    * NEW_IMAGE - The entire item, as it appears after it was modified, is
		//    written to the stream.
		//
		//    * OLD_IMAGE - The entire item, as it appeared before it was modified,
		//    is written to the stream.
		//
		//    * NEW_AND_OLD_IMAGES - Both the new and the old item images of the item
		//    are written to the stream.
		ViewType string `validate:"oneof=KEYS_ONLY NEW_IMAGE OLD_IMAGE NEW_AND_OLD_IMAGES"`
	} `func:"input"`

	// The name of the table to create.
	TableName string `func:"input" validate:"min=3"`

	// A list of key-value pairs to label the table. For more information, see
	// [Tagging for
	// DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Tagging.html).
	Tags []struct {
		// The key of the tag. Tag keys are case sensitive. Each DynamoDB table
		// can only have up to one tag with the same key. If you try to add an
		// existing tag (same key), the existing tag value will be updated to
		// the new value.
		Key string `validate:"min=1"`

		// The value of the tag. Tag values are case-sensitive and can be null.
		Value string
	} `func:"input" name:"tag"`

	Region string `func:"input"`

	// Outputs

	// RFC3339 formatted date and time for when the table was created.
	CreatedTime string `func:"output"`

	// The Amazon Resource Name (ARN) that uniquely identifies the table.
	TableARN string `func:"output"`

	// Unique identifier for the table for which the backup was created.
	TableID string `func:"output"`

	dynamoDBService
}

// Create creates a new DynamoDB table.
func (p *DynamoDBTable) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &dynamodb.CreateTableInput{}

	input.AttributeDefinitions = make([]dynamodb.AttributeDefinition, len(p.Attributes))
	for i, a := range p.Attributes {
		input.AttributeDefinitions[i] = dynamodb.AttributeDefinition{
			AttributeName: aws.String(a.Name),
			AttributeType: dynamodb.ScalarAttributeType(a.Type),
		}
	}

	input.BillingMode = dynamodb.BillingMode(p.BillingMode)

	input.GlobalSecondaryIndexes = make([]dynamodb.GlobalSecondaryIndex, len(p.GlobalSecondaryIndexes))
	for i, g := range p.GlobalSecondaryIndexes {
		gsi := dynamodb.GlobalSecondaryIndex{}
		gsi.IndexName = aws.String(g.Name)
		if len(g.KeySchema) > 0 {
			gsi.KeySchema = make([]dynamodb.KeySchemaElement, len(g.KeySchema))
			for j, ks := range g.KeySchema {
				gsi.KeySchema[j] = dynamodb.KeySchemaElement{
					AttributeName: aws.String(ks.Name),
					KeyType:       dynamodb.KeyType(ks.Type),
				}
			}
		}
		gsi.Projection = &dynamodb.Projection{
			NonKeyAttributes: g.Projection.NonKeyAttributes,
			ProjectionType:   dynamodb.ProjectionType(g.Projection.Type),
		}
		if g.ProvisionedThroughput != nil {
			gsi.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(g.ProvisionedThroughput.ReadCapacityUnits),
				WriteCapacityUnits: aws.Int64(g.ProvisionedThroughput.WriteCapacityUnits),
			}
		}
		input.GlobalSecondaryIndexes[i] = gsi
	}

	if len(p.KeySchema) > 0 {
		input.KeySchema = make([]dynamodb.KeySchemaElement, len(p.KeySchema))
		for j, ks := range p.KeySchema {
			input.KeySchema[j] = dynamodb.KeySchemaElement{
				AttributeName: aws.String(ks.Name),
				KeyType:       dynamodb.KeyType(ks.Type),
			}
		}
	}

	if len(p.LocalSecondaryIndexes) > 0 {
		input.LocalSecondaryIndexes = make([]dynamodb.LocalSecondaryIndex, len(p.LocalSecondaryIndexes))
		for i, l := range p.LocalSecondaryIndexes {
			lsi := dynamodb.LocalSecondaryIndex{}
			lsi.IndexName = aws.String(l.Name)
			if len(l.KeySchema) > 0 {
				lsi.KeySchema = make([]dynamodb.KeySchemaElement, len(l.KeySchema))
				for j, ks := range l.KeySchema {
					lsi.KeySchema[j] = dynamodb.KeySchemaElement{
						AttributeName: aws.String(ks.Name),
						KeyType:       dynamodb.KeyType(ks.Type),
					}
				}
			}
			lsi.Projection = &dynamodb.Projection{
				NonKeyAttributes: l.Projection.NonKeyAttributes,
				ProjectionType:   dynamodb.ProjectionType(l.Projection.Type),
			}
			input.LocalSecondaryIndexes[i] = lsi
		}
	}

	if p.ProvisionedThroughput != nil {
		input.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(p.ProvisionedThroughput.ReadCapacityUnits),
			WriteCapacityUnits: aws.Int64(p.ProvisionedThroughput.WriteCapacityUnits),
		}
	}

	if p.SSE != nil {
		input.SSESpecification = &dynamodb.SSESpecification{
			Enabled:        p.SSE.Enabled,
			KMSMasterKeyId: p.SSE.KMSMasterKeyID,
			SSEType:        dynamodb.SSETypeKms, // Only one supported
			// SSEType:        dynamodb.SSEType(p.SSE.Type),
		}
	}
	if p.Stream != nil {
		input.StreamSpecification = &dynamodb.StreamSpecification{
			StreamEnabled:  p.Stream.Enabled,
			StreamViewType: dynamodb.StreamViewType(p.Stream.ViewType),
		}
	}

	input.TableName = aws.String(p.TableName)

	if len(p.Tags) > 0 {
		input.Tags = make([]dynamodb.Tag, len(p.Tags))
		for i, t := range p.Tags {
			input.Tags[i] = dynamodb.Tag{
				Key:   aws.String(t.Key),
				Value: aws.String(t.Value),
			}
		}
	}

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreateTableRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	desc := resp.CreateTableOutput.TableDescription
	p.CreatedTime = desc.CreationDateTime.Format(time.RFC3339)
	p.TableARN = *desc.TableArn
	p.TableID = *desc.TableId

	return nil
}

// Delete deletes the DynamoDB table.
func (p *DynamoDBTable) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(p.TableName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteTableRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update updates the DynamoDB table.
func (p *DynamoDBTable) Update(ctx context.Context, r *resource.UpdateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	prev := r.Previous.(*DynamoDBTable)

	input := &dynamodb.UpdateTableInput{}

	input.AttributeDefinitions = make([]dynamodb.AttributeDefinition, len(p.Attributes))
	for i, a := range p.Attributes {
		input.AttributeDefinitions[i] = dynamodb.AttributeDefinition{
			AttributeName: aws.String(a.Name),
			AttributeType: dynamodb.ScalarAttributeType(a.Type),
		}
	}

	input.BillingMode = dynamodb.BillingMode(p.BillingMode)

	if !cmp.Equal(p.GlobalSecondaryIndexes, prev.GlobalSecondaryIndexes) {
		// TODO: Compute GSI diff
		return backoff.Permanent(fmt.Errorf("updating global secondary indexes is not yet supported"))
	}

	if p.ProvisionedThroughput != nil {
		input.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(p.ProvisionedThroughput.ReadCapacityUnits),
			WriteCapacityUnits: aws.Int64(p.ProvisionedThroughput.WriteCapacityUnits),
		}
	}

	if p.SSE != nil {
		input.SSESpecification = &dynamodb.SSESpecification{
			Enabled:        p.SSE.Enabled,
			KMSMasterKeyId: p.SSE.KMSMasterKeyID,
			SSEType:        dynamodb.SSETypeKms, // Only one supported
			// SSEType:        dynamodb.SSEType(p.SSE.Type),
		}
	}
	if p.Stream != nil {
		input.StreamSpecification = &dynamodb.StreamSpecification{
			StreamEnabled:  p.Stream.Enabled,
			StreamViewType: dynamodb.StreamViewType(p.Stream.ViewType),
		}
	}

	input.TableName = aws.String(prev.TableName)

	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.UpdateTableRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	_ = resp

	return nil
}
