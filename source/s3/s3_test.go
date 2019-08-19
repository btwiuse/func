// +build integration

package s3

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/func/func/source"
)

func TestStorage(t *testing.T) {
	bucket := "test-bucket"

	cli := getAPI(t)
	cleanup := makeBucket(t, cli, bucket)
	defer cleanup()

	s := &Storage{
		Bucket:          bucket,
		UploadURLExpiry: 5 * time.Minute,
		Client:          cli,
	}

	ctx := context.Background()

	name := "file.txt"
	data := []byte("foo")

	h := md5.Sum(data)
	md5 := base64.StdEncoding.EncodeToString(h[:])

	// File does not exist
	has, err := s.Has(ctx, name)
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if has {
		t.Fatalf("Has() got = %t, want = %t", has, false)
	}

	// Create upload
	u, err := s.NewUpload(source.UploadConfig{
		Filename:      name,
		ContentMD5:    md5,
		ContentLength: len(data),
	})
	if err != nil {
		t.Fatalf("NewUpload() error = %v", err)
	}

	// Upload data
	req, err := http.NewRequest(http.MethodPut, u.URL, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	for k, v := range u.Headers {
		req.Header.Add(k, v)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		if dumped, err := httputil.DumpResponse(resp, true); err == nil {
			t.Log(string(dumped))
		}
		t.Fatalf("Upload statusCode = %d", resp.StatusCode)
	}

	// File should now exist
	has, err = s.Has(ctx, name)
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if !has {
		t.Fatalf("Has() got = %t, want = %t", has, true)
	}

	// Read back file
	f, err := s.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	got, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	want := []byte("foo")
	if !bytes.Equal(got, want) {
		t.Errorf("Stored data does not match\nGot\n%s\nWant\n%s", hex.Dump(got), hex.Dump(want))
	}
}

func getAPI(t *testing.T) *s3.Client {
	t.Helper()

	accessKey := os.Getenv("TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("TEST_S3_SECRET_KEY")
	region := os.Getenv("TEST_S3_REGION")
	endpoint := os.Getenv("TEST_S3_ENDPOINT")

	if accessKey == "" {
		t.Fatal("TEST_S3_ACCESS_KEY not set")
	}
	if secretKey == "" {
		t.Fatal("TEST_S3_SECRET_KEY not set")
	}
	if region == "" {
		t.Fatal("TEST_S3_REGION not set")
	}

	cfgs := external.Configs{
		external.WithRegion(region),
		external.WithCredentialsValue(aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
		}),
	}
	cfg, err := cfgs.ResolveAWSConfig(external.DefaultAWSConfigResolvers)
	if err != nil {
		t.Fatal(err)
	}

	cli := s3.New(cfg)

	if endpoint != "" {
		cli.Config.EndpointResolver = aws.ResolveWithEndpointURL(endpoint)
		cli.ForcePathStyle = true
	}

	return cli
}

func makeBucket(t *testing.T, cli s3iface.ClientAPI, name string) func() {
	t.Helper()

	ctx := context.Background()

	_, err := cli.CreateBucketRequest(&s3.CreateBucketInput{Bucket: aws.String(name)}).Send(ctx)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	cleanup := func() {
		res, err := cli.ListObjectsV2Request(&s3.ListObjectsV2Input{
			Bucket: aws.String(name),
		}).Send(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ids := make([]s3.ObjectIdentifier, len(res.Contents))
		for i, c := range res.Contents {
			ids[i] = s3.ObjectIdentifier{
				Key: c.Key,
			}
		}

		if _, err = cli.DeleteObjectsRequest(&s3.DeleteObjectsInput{
			Bucket: aws.String(name),
			Delete: &s3.Delete{
				Objects: ids,
			},
		}).Send(ctx); err != nil {
			t.Fatal(err)
		}

		if _, err = cli.DeleteBucketRequest(&s3.DeleteBucketInput{Bucket: aws.String(name)}).Send(ctx); err != nil {
			t.Fatal(err)
		}
	}

	return cleanup
}
