package s3

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/func/func/source"
	"github.com/pkg/errors"
)

// S3 stores data in an AWS S3 bucket.
type S3 struct {
	bucket          string        // S3 Bucket is the bucket to use.
	uploadURLExpiry time.Duration // Expiry time for signed upload URLs.
	cli             s3iface.ClientAPI
}

// New creates a new S3 storage client.
func New(cfg aws.Config, bucket string, uploadExpiry time.Duration) *S3 {
	return &S3{
		bucket:          bucket,
		uploadURLExpiry: uploadExpiry,
		cli:             s3.New(cfg),
	}
}

// Has returns true if the given filename exists in the bucket.
func (s *S3) Has(ctx context.Context, filename string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	}

	req := s.cli.HeadObjectRequest(input)
	_, err := req.Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, errors.Wrap(err, "send request")
	}

	return true, nil
}

// Get returns a reader for a file in the bucket.
func (s *S3) Get(ctx context.Context, filename string) (io.ReadCloser, error) {
	req := s.cli.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})
	res, err := req.Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "send request")
	}
	return res.Body, nil
}

// NewUpload creates a new upload url that allows a user to upload a file to
// the bucket.
//
// The uploaded file must match the provided ContentMD5.
func (s *S3) NewUpload(config source.UploadConfig) (*source.UploadURL, error) {
	req := s.cli.PutObjectRequest(&s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(config.Filename),
		ContentLength: aws.Int64(int64(config.ContentLength)),
		ContentMD5:    aws.String(config.ContentMD5),
	})

	presigned, err := req.Presign(s.uploadURLExpiry)
	if err != nil {
		return nil, errors.Wrap(err, "presign upload url")
	}

	url := &source.UploadURL{
		URL: presigned,
		Headers: map[string]string{
			"Content-MD5":    config.ContentMD5,
			"Content-Length": strconv.Itoa(config.ContentLength),
		},
	}

	return url, nil
}
