package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/proarash/ecommerce-server/internal/config"
)

var ErrStorageUnavailable = errors.New("object storage is not configured")

type Client struct {
	mc        *minio.Client
	bucket    string
	publicURL string
}

func NewClient(ctx context.Context, cfg *config.EnvConfig) *Client {
	c := &Client{bucket: cfg.MinioBucket}
	if cfg.MinioEndpoint == "" {
		log.Println("minio: MINIO_ENDPOINT not set, uploads disabled")
		return c
	}
	useSSL := cfg.MinioUseSSL == "true"
	mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Println("minio:", err)
		return c
	}
	if err := mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
		exists, errExists := mc.BucketExists(ctx, c.bucket)
		if errExists != nil || !exists {
			log.Println("minio:", err)
			return c
		}
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, c.bucket)
	if err := mc.SetBucketPolicy(ctx, c.bucket, policy); err != nil {
		log.Println("minio: set bucket policy:", err)
	}
	c.mc = mc
	c.publicURL = strings.TrimRight(cfg.MinioPublicURL, "/")
	if c.publicURL == "" {
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		c.publicURL = fmt.Sprintf("%s://%s", scheme, cfg.MinioEndpoint)
	}
	return c
}

func (c *Client) Upload(ctx context.Context, objectName string, r io.Reader, size int64, contentType string) (string, error) {
	if c.mc == nil {
		return "", ErrStorageUnavailable
	}
	_, err := c.mc.PutObject(ctx, c.bucket, objectName, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", c.publicURL, c.bucket, objectName), nil
}
