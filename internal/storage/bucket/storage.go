package bucket

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3     *s3.Client
	bucket string
	prefix string
}

func NewClient(ctx context.Context, cfg *S3Conf) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	if cfg.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(cfg.Endpoint)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &Client{
		s3:     client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

func (c *Client) Upload(ctx context.Context, path string, r io.Reader, payloadBytesLen int64) error {
	key := c.prefix + path
	if _, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		ContentLength: aws.Int64(payloadBytesLen),
		Key:           aws.String(key),
		Body:          r,
	}); err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (c *Client) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	key := c.prefix + path

	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return out.Body, nil
}

func (c *Client) DownloadBytes(ctx context.Context, path string) ([]byte, error) {
	r, err := c.Download(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("download object: %w", err)
	}
	defer r.Close() //nolint:errcheck // it ok

	b, err := io.ReadAll(io.LimitReader(r, c.cfg.MaxUploadSizeBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read object body: %w", err)
	}
	if int64(len(b)) > c.cfg.MaxUploadSizeBytes {
		return nil, fmt.Errorf("object exceeds max allowed size: %d", c.cfg.MaxUploadSizeBytes)
	}
	return b, nil
}

func (c *Client) Delete(ctx context.Context, path string) error {
	key := c.prefix + path
	if _, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}
