package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	common_config "github.com/martinshumberto/sync-manager/common/config"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSConfig holds configuration for GCS
type GCSConfig struct {
	ProjectID       string
	Bucket          string
	CredentialsFile string
}

// NewGCSConfigFromCommon converts a common.GCSConfig to storage.GCSConfig
func NewGCSConfigFromCommon(commonCfg *common_config.GCSConfig) *GCSConfig {
	return &GCSConfig{
		ProjectID:       commonCfg.ProjectID,
		Bucket:          commonCfg.Bucket,
		CredentialsFile: commonCfg.CredentialsFile,
	}
}

// GCSStorage implements the Storage interface using Google Cloud Storage
type GCSStorage struct {
	client *storage.Client
	bucket string
	config *GCSConfig
}

// GetProvider returns the storage provider type
func (g *GCSStorage) GetProvider() StorageProvider {
	return ProviderGCS
}

// NewGCSStorage creates a new GCS storage client
func NewGCSStorage(cfg *GCSConfig) (*GCSStorage, error) {
	ctx := context.Background()
	var client *storage.Client
	var err error

	if cfg.CredentialsFile != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(cfg.CredentialsFile))
	} else {
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	bucket := client.Bucket(cfg.Bucket)
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket: %w", err)
	}

	return &GCSStorage{
		client: client,
		bucket: cfg.Bucket,
		config: cfg,
	}, nil
}

// UploadFile uploads a file to GCS
func (g *GCSStorage) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) (string, error) {
	key = strings.TrimPrefix(key, "/")

	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(key)
	w := obj.NewWriter(ctx)

	w.Metadata = metadata

	if _, err := io.Copy(w, reader); err != nil {
		w.Close()
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize upload: %w", err)
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %w", err)
	}

	log.Debug().
		Str("bucket", g.bucket).
		Str("key", key).
		Int64("generation", attrs.Generation).
		Msg("Uploaded file to GCS")

	return fmt.Sprintf("%d", attrs.Generation), nil
}

// DownloadFile downloads a file from GCS
func (g *GCSStorage) DownloadFile(ctx context.Context, key string, writer io.Writer, versionID string) (map[string]string, error) {
	key = strings.TrimPrefix(key, "/")

	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(key)

	if versionID != "" {
		var generation int64
		fmt.Sscanf(versionID, "%d", &generation)
		if generation > 0 {
			obj = obj.Generation(generation)
		}
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer r.Close()

	if _, err := io.Copy(writer, r); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	log.Debug().
		Str("bucket", g.bucket).
		Str("key", key).
		Int64("generation", attrs.Generation).
		Msg("Downloaded file from GCS")

	return attrs.Metadata, nil
}

// DeleteFile deletes a file from GCS
func (g *GCSStorage) DeleteFile(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")

	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(key)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	log.Debug().
		Str("bucket", g.bucket).
		Str("key", key).
		Msg("Deleted file from GCS")

	return nil
}

// ListFiles lists files in GCS with the given prefix
func (g *GCSStorage) ListFiles(ctx context.Context, prefix string) ([]FileInfo, error) {
	prefix = strings.TrimPrefix(prefix, "/")

	bucket := g.client.Bucket(g.bucket)

	var files []FileInfo
	it := bucket.Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}

		files = append(files, FileInfo{
			Key:          attrs.Name,
			Size:         attrs.Size,
			LastModified: attrs.Updated,
			ETag:         fmt.Sprintf("%d", attrs.Generation),
		})
	}

	log.Debug().
		Str("bucket", g.bucket).
		Str("prefix", prefix).
		Int("count", len(files)).
		Msg("Listed files from GCS")

	return files, nil
}

// FileExists checks if a file exists in GCS
func (g *GCSStorage) FileExists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(key)

	_, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
}
