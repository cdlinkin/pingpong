package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(
	ctx context.Context,
	endpoint, accessKey, secretKey, bucket string,
	useSSL bool,
) (*S3, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"create s3 client: %w",
			err,
		)
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf(
			"check bucket: %w",
			err,
		)
	}

	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf(
				"create bucket: %w",
				err,
			)
		}
	}

	return &S3{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *S3) Remove(
	ctx context.Context,
	key string,
) error {
	if err := checkKey(key); err != nil {
		return fmt.Errorf(
			"key is not validate: %w",
			err,
		)
	}

	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf(
			"remove object: %w",
			err,
		)
	}

	return nil
}

func (s *S3) Open(
	ctx context.Context,
	key string,
) (io.ReadSeekCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, fmt.Errorf(
			"key is not validate: %w",
			err,
		)
	}

	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"get object: %w",
			err,
		)
	}
	_, err = obj.Stat()
	if err != nil {
		obj.Close()
		return nil, fmt.Errorf(
			"object stat: %w",
			err,
		)
	}

	return obj, nil
}

func (s *S3) Save(
	ctx context.Context,
	r io.Reader,
) (string, int64, error) {
	key, err := newKey()
	if err != nil {
		return "", 0, fmt.Errorf(
			"generate key: %w",
			err,
		)
	}

	info, err := s.client.PutObject(ctx, s.bucket, key, r, -1, minio.PutObjectOptions{})
	if err != nil {
		return "", 0, fmt.Errorf(
			"put object: %w",
			err,
		)
	}

	return key, info.Size, nil
}
