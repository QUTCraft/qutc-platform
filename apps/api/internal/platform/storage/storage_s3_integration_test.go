//go:build integration

package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestS3RoundTripAgainstMinIO(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	accessKey := os.Getenv("S3_TEST_ACCESS_KEY")
	secretKey := os.Getenv("S3_TEST_SECRET_KEY")
	if endpoint == "" || accessKey == "" || secretKey == "" {
		t.Skip("S3_TEST_ENDPOINT, S3_TEST_ACCESS_KEY and S3_TEST_SECRET_KEY are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := NewS3(ctx, Config{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    valueOrDefault(os.Getenv("S3_TEST_BUCKET"), "qutcraft-media-test"),
		Region:    "us-east-1",
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	key := "integration/" + uuid.NewString() + ".txt"
	source := []byte("qutcraft-minio-integration")
	written, err := store.Put(ctx, key, bytes.NewReader(source), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if written != int64(len(source)) {
		t.Fatalf("written = %d, want %d", written, len(source))
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), key) })

	reader, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	actual, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || !bytes.Equal(actual, source) {
		t.Fatalf("read = %q, %v", actual, err)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Open after Delete = %v, want ErrNotFound", err)
	}
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
