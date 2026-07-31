package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var ErrNotFound = errors.New("storage object not found")

type Store interface {
	Driver() string
	Put(context.Context, string, io.Reader, string) (int64, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Config struct {
	Driver    string
	LocalRoot string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

func New(ctx context.Context, cfg Config) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", "local":
		return NewLocal(cfg.LocalRoot)
	case "s3":
		return NewS3(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.Driver)
	}
}

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	if strings.TrimSpace(root) == "" {
		root = "/tmp/qutcraft-uploads"
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage root: %w", err)
	}
	if err := os.MkdirAll(absoluteRoot, 0o750); err != nil {
		return nil, fmt.Errorf("create local storage root: %w", err)
	}
	return &Local{root: filepath.Clean(absoluteRoot)}, nil
}

func (l *Local) Driver() string {
	return "local"
}

func (l *Local) Put(_ context.Context, key string, source io.Reader, _ string) (int64, error) {
	path, err := l.resolve(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return 0, fmt.Errorf("create object directory: %w", err)
	}
	target, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return 0, fmt.Errorf("create local object: %w", err)
	}
	written, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		if copyErr != nil {
			return 0, fmt.Errorf("write local object: %w", copyErr)
		}
		return 0, fmt.Errorf("close local object: %w", closeErr)
	}
	return written, nil
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open local object: %w", err)
	}
	return file, nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	path, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete local object: %w", err)
	}
	return nil
}

func (l *Local) resolve(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("storage key is empty")
	}
	path := key
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.root, filepath.FromSlash(key))
	}
	path = filepath.Clean(path)
	relative, err := filepath.Rel(l.root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage key escapes local root")
	}
	return path, nil
}

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	for {
		err = ensureS3Bucket(ctx, client, cfg.Bucket, cfg.Region)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("initialize S3 bucket: %w", err)
		case <-time.After(500 * time.Millisecond):
		}
	}
	return &S3{client: client, bucket: cfg.Bucket}, nil
}

func ensureS3Bucket(ctx context.Context, client *minio.Client, bucket, region string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check S3 bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
		response := minio.ToErrorResponse(err)
		if response.Code == "BucketAlreadyOwnedByYou" || response.Code == "BucketAlreadyExists" {
			return nil
		}
		return fmt.Errorf("create S3 bucket: %w", err)
	}
	return nil
}

func (s *S3) Driver() string {
	return "s3"
}

func (s *S3) Put(ctx context.Context, key string, source io.Reader, contentType string) (int64, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, source, -1, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return 0, fmt.Errorf("put S3 object: %w", err)
	}
	return info.Size, nil
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, classifyS3Error(err)
	}
	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, classifyS3Error(err)
	}
	return object, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return classifyS3Error(err)
	}
	return nil
}

func classifyS3Error(err error) error {
	response := minio.ToErrorResponse(err)
	switch response.Code {
	case "NoSuchKey", "NoSuchObject", "NoSuchBucket", "NotFound":
		return ErrNotFound
	default:
		return fmt.Errorf("S3 operation failed: %w", err)
	}
}
