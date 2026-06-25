package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const presignedExpiry = 24 * time.Hour

type Client struct {
	client     *minio.Client
	bucketName string
}

func NewClient() (*Client, error) {
	endpoint := os.Getenv("BUCKET_ENDPOINT")
	accessKey := os.Getenv("BUCKET_ACCESS_KEY")
	secretKey := os.Getenv("BUCKET_SECRET_KEY")
	bucketName := os.Getenv("BUCKET_NAME")
	region := os.Getenv("BUCKET_REGION")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucketName == "" {
		return nil, fmt.Errorf("BUCKET_ENDPOINT, BUCKET_ACCESS_KEY, BUCKET_SECRET_KEY, and BUCKET_NAME must be set")
	}
	if region == "" {
		region = "us-east-1"
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &Client{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (c *Client) Upload(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error) {
	_, err := c.client.PutObject(ctx, c.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}
	return objectName, nil
}

func (c *Client) GetURL(ctx context.Context, objectName string) (string, error) {
	u, err := c.client.PresignedGetObject(ctx, c.bucketName, objectName, presignedExpiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return u.String(), nil
}

func (c *Client) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, c.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return obj, nil
}

func (c *Client) Delete(ctx context.Context, objectName string) error {
	err := c.client.RemoveObject(ctx, c.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
