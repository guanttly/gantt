// pkg/adapter/minio.go
package adapter

import (
	"bytes"
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	Client *minio.Client
}

var ctx = context.Background()

func NewMinIO(endpoint, accessKey, secretKey string) (*MinIO, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, err
	}
	return &MinIO{Client: client}, nil
}

func (m *MinIO) UploadFile(bucketName, objectName string, data []byte) error {
	// 检查桶是否存在，不存在则创建
	exists, err := m.Client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		err = m.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}
	_, err = m.Client.PutObject(ctx, bucketName, objectName,
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return err
}

func (m *MinIO) GetFile(bucketName, objectName string) ([]byte, error) {
	obj, err := m.Client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (m *MinIO) ListFiles(bucketName string) ([]string, error) {
	var files []string
	objectCh := m.Client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true})
	for obj := range objectCh {
		if obj.Err != nil {
			return nil, obj.Err
		}
		files = append(files, obj.Key)
	}
	return files, nil
}

func (m *MinIO) DeleteFile(bucketName, objectName string) error {
	return m.Client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

func (m *MinIO) ToErrorResponse(err error) minio.ErrorResponse {
	if err != nil {
		if minioErr, ok := err.(minio.ErrorResponse); ok {
			return minioErr
		}
	}
	return minio.ErrorResponse{Code: "UnknownError", Message: err.Error()}
}
