package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	common_config "github.com/martinshumberto/sync-manager/common/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// MinioConfig holds configuration for MinIO
type MinioConfig struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// NewMinioConfigFromCommon converts a common.MinioConfig to storage.MinioConfig
func NewMinioConfigFromCommon(commonCfg *common_config.MinioConfig) *MinioConfig {
	return &MinioConfig{
		Endpoint:  commonCfg.Endpoint,
		Region:    commonCfg.Region,
		Bucket:    commonCfg.Bucket,
		AccessKey: commonCfg.AccessKey,
		SecretKey: commonCfg.SecretKey,
		UseSSL:    commonCfg.UseSSL,
	}
}

// MinioStorage implements the Storage interface using MinIO
type MinioStorage struct {
	client *minio.Client
	bucket string
	config *MinioConfig
}

// GetProvider returns the storage provider type
func (m *MinioStorage) GetProvider() StorageProvider {
	return ProviderMinio
}

// NewMinioStorage creates a new MinIO storage client
func NewMinioStorage(cfg *MinioConfig) (*MinioStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	exists, err := client.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = client.MakeBucket(context.Background(), cfg.Bucket, minio.MakeBucketOptions{
			Region: cfg.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Info().Str("bucket", cfg.Bucket).Msg("Created MinIO bucket")
	}

	return &MinioStorage{
		client: client,
		bucket: cfg.Bucket,
		config: cfg,
	}, nil
}

// UploadFile uploads a file to MinIO
func (m *MinioStorage) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) (string, error) {
	key = strings.TrimPrefix(key, "/")

	userMetadata := make(map[string]string)
	for k, v := range metadata {
		userMetadata[k] = v
	}

	info, err := m.client.PutObject(ctx, m.bucket, key, reader, -1, minio.PutObjectOptions{
		UserMetadata: userMetadata,
		ContentType:  metadata["content_type"],
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	log.Debug().
		Str("bucket", m.bucket).
		Str("key", key).
		Str("etag", info.ETag).
		Msg("Uploaded file to MinIO")

	// Return the ETag as version ID
	return info.ETag, nil
}

// DownloadFile downloads a file from MinIO
func (m *MinioStorage) DownloadFile(ctx context.Context, key string, writer io.Writer, versionID string) (map[string]string, error) {
	key = strings.TrimPrefix(key, "/")

	opts := minio.GetObjectOptions{}
	if versionID != "" {
		opts.VersionID = versionID
	}

	obj, err := m.client.GetObject(ctx, m.bucket, key, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	if _, err := io.Copy(writer, obj); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	metadata := make(map[string]string)
	for k, v := range stat.UserMetadata {
		metadata[k] = v
	}

	log.Debug().
		Str("bucket", m.bucket).
		Str("key", key).
		Str("etag", stat.ETag).
		Msg("Downloaded file from MinIO")

	return metadata, nil
}

// DeleteFile deletes a file from MinIO
func (m *MinioStorage) DeleteFile(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")

	err := m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	log.Debug().
		Str("bucket", m.bucket).
		Str("key", key).
		Msg("Deleted file from MinIO")

	return nil
}

// ListFiles lists files in MinIO with the given prefix
func (m *MinioStorage) ListFiles(ctx context.Context, prefix string) ([]FileInfo, error) {
	prefix = strings.TrimPrefix(prefix, "/")

	objectCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var files []FileInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}

		files = append(files, FileInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         strings.Trim(object.ETag, "\""),
		})
	}

	log.Debug().
		Str("bucket", m.bucket).
		Str("prefix", prefix).
		Int("count", len(files)).
		Msg("Listed files from MinIO")

	return files, nil
}

// FileExists checks if a file exists in MinIO
func (m *MinioStorage) FileExists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	_, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
}
