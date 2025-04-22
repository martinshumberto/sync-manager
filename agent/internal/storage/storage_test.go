package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileInfo(t *testing.T) {
	info := FileInfo{
		Key:  "test-key",
		Size: 1024,
		ETag: "test-etag",
	}

	assert.Equal(t, "test-key", info.Key)
	assert.Equal(t, int64(1024), info.Size)
	assert.Equal(t, "test-etag", info.ETag)
}

func TestS3Config(t *testing.T) {
	cfg := S3Config{
		Endpoint:  "localhost:9000",
		Region:    "us-east-1",
		Bucket:    "test-bucket",
		AccessKey: "test-access",
		SecretKey: "test-secret",
		UseSSL:    true,
		PathStyle: false,
	}

	assert.Equal(t, "localhost:9000", cfg.Endpoint)
	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "test-bucket", cfg.Bucket)
	assert.Equal(t, "test-access", cfg.AccessKey)
	assert.Equal(t, "test-secret", cfg.SecretKey)
	assert.True(t, cfg.UseSSL)
	assert.False(t, cfg.PathStyle)
}
