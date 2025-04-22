package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	common_config "github.com/martinshumberto/sync-manager/common/config"
	"github.com/rs/zerolog/log"
)

// S3Config holds configuration for S3
type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	PathStyle bool
}

// NewS3ConfigFromCommon converts a common.S3Config to storage.S3Config
func NewS3ConfigFromCommon(commonCfg *common_config.S3Config) *S3Config {
	return &S3Config{
		Endpoint:  commonCfg.Endpoint,
		Region:    commonCfg.Region,
		Bucket:    commonCfg.Bucket,
		AccessKey: commonCfg.AccessKey,
		SecretKey: commonCfg.SecretKey,
		UseSSL:    commonCfg.UseSSL,
		PathStyle: commonCfg.PathStyle,
	}
}

// S3Storage implements the Storage interface using S3
type S3Storage struct {
	client *s3.Client
	bucket string
	config *S3Config
}

// GetProvider returns the storage provider type
func (s *S3Storage) GetProvider() StorageProvider {
	return ProviderS3
}

// NewS3Storage creates a new S3 storage client
func NewS3Storage(cfg *S3Config) (*S3Storage, error) {
	var resolver aws.EndpointResolverWithOptions
	var awsConfig aws.Config
	var err error

	if cfg.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				protocol := "https"
				if !cfg.UseSSL {
					protocol = "http"
				}
				return aws.Endpoint{
					URL:               fmt.Sprintf("%s://%s", protocol, cfg.Endpoint),
					SigningRegion:     cfg.Region,
					HostnameImmutable: cfg.PathStyle,
				}, nil
			}
			// Fallback to default resolver
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		resolver = customResolver

		awsConfig, err = awsconfig.LoadDefaultConfig(
			context.Background(),
			awsconfig.WithRegion(cfg.Region),
			awsconfig.WithEndpointResolverWithOptions(resolver),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			),
		)
	} else {
		awsConfig, err = awsconfig.LoadDefaultConfig(
			context.Background(),
			awsconfig.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = cfg.PathStyle
	})

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
		config: cfg,
	}, nil
}

// UploadFile uploads a file to S3
func (s *S3Storage) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) (string, error) {
	key = strings.TrimPrefix(key, "/")

	awsMetadata := make(map[string]string)
	for k, v := range metadata {
		awsMetadata[k] = v
	}

	output, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: awsMetadata,
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	log.Debug().
		Str("bucket", s.bucket).
		Str("key", key).
		Str("version_id", aws.ToString(output.VersionId)).
		Msg("Uploaded file to S3")

	return aws.ToString(output.VersionId), nil
}

// DownloadFile downloads a file from S3
func (s *S3Storage) DownloadFile(ctx context.Context, key string, writer io.Writer, versionID string) (map[string]string, error) {
	key = strings.TrimPrefix(key, "/")

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer output.Body.Close()

	if _, err := io.Copy(writer, output.Body); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	metadata := make(map[string]string)
	for k, v := range output.Metadata {
		metadata[k] = v
	}

	log.Debug().
		Str("bucket", s.bucket).
		Str("key", key).
		Str("version_id", aws.ToString(output.VersionId)).
		Msg("Downloaded file from S3")

	return metadata, nil
}

// DeleteFile deletes a file from S3
func (s *S3Storage) DeleteFile(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	log.Debug().
		Str("bucket", s.bucket).
		Str("key", key).
		Msg("Deleted file from S3")

	return nil
}

// ListFiles lists files in S3 with the given prefix
func (s *S3Storage) ListFiles(ctx context.Context, prefix string) ([]FileInfo, error) {
	prefix = strings.TrimPrefix(prefix, "/")

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	var files []FileInfo
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		for _, obj := range page.Contents {
			files = append(files, FileInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: *obj.LastModified,
				ETag:         strings.Trim(aws.ToString(obj.ETag), "\""),
			})
		}
	}

	log.Debug().
		Str("bucket", s.bucket).
		Str("prefix", prefix).
		Int("count", len(files)).
		Msg("Listed files from S3")

	return files, nil
}

// FileExists checks if a file exists in S3
func (s *S3Storage) FileExists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
}
