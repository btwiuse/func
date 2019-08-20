// +build integration

package dynamodb

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/func/func/resource"
	"github.com/func/func/storage/testsuite"
)

func TestDynamoDB(t *testing.T) {
	testsuite.Run(t, testsuite.Config{
		New: func(t *testing.T, types map[string]reflect.Type) (testsuite.Target, func()) {
			endpoint := os.Getenv("TEST_DYNAMODB_ENDPOINT")

			if endpoint == "" {
				t.Fatal("TEST_DYNAMODB_ENDPOINT not set")
			}

			cfg := defaults.Config()
			cfg.Region = "local"
			cfg.EndpointResolver = aws.ResolveWithEndpointURL(endpoint)
			cfg.Credentials = aws.NewStaticCredentialsProvider("local", "local", "")

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			ddb := New(cfg, "test-test", &resource.Registry{Types: types})
			if err := ddb.CreateTable(ctx, 1, 1); err != nil {
				t.Log("Maybe DynamoDB local is not running?")
				t.Fatalf("Create resource table: %v", err)
			}

			cleanup := func() {
				cli := dynamodb.New(cfg)
				_, err := cli.DeleteTableRequest(&dynamodb.DeleteTableInput{
					TableName: aws.String(ddb.TableName),
				}).Send(context.Background())
				if err != nil {
					t.Fatalf("Delete DynamoDB table: %v", err)
				}
			}

			return ddb, cleanup
		},
	})
}
