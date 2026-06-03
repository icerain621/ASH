package artifactstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Profile struct {
	Kind              string `json:"kind"`
	Ready             bool   `json:"ready"`
	URI               string `json:"uri,omitempty"`
	ObjectStore       bool   `json:"objectStore"`
	SupportsSignedURL bool   `json:"supportsSignedUrl"`
	Error             string `json:"error,omitempty"`
}

type ObjectRef struct {
	URI         string
	Key         string
	ContentType string
	SizeBytes   int64
}

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) (*ObjectRef, error)
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

func New(kind, dataDir string) Store {
	switch normalizeKind(kind) {
	case "fs":
		return NewFSStore(filepath.Join(dataDir, "object_store"))
	case "s3", "s3-compatible":
		cfg, profile := loadS3Config(normalizeKind(kind))
		if !profile.Ready {
			return UnsupportedStore{Kind: profile.Kind, Reason: profile.Error}
		}
		return NewS3Store(cfg)
	default:
		return UnsupportedStore{Kind: kind}
	}
}

func Describe(kind, dataDir string) Profile {
	switch normalizeKind(kind) {
	case "fs":
		root := filepath.ToSlash(filepath.Join(dataDir, "object_store"))
		return Profile{Kind: "fs", Ready: true, URI: "fs://" + root, SupportsSignedURL: true}
	case "s3", "s3-compatible":
		_, profile := loadS3Config(normalizeKind(kind))
		return profile
	default:
		return Profile{Kind: normalizeKind(kind), Ready: false, Error: "unsupported artifact store"}
	}
}

func normalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "", "local", "filesystem":
		return "fs"
	default:
		return kind
	}
}

type FSStore struct {
	root string
}

func NewFSStore(root string) *FSStore {
	return &FSStore{root: root}
}

func (s *FSStore) Put(_ context.Context, key string, r io.Reader, contentType string) (*ObjectRef, error) {
	cleanKey := cleanObjectKey(key)
	path := filepath.Join(s.root, cleanKey)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return nil, err
	}
	return &ObjectRef{
		URI: "fs://" + filepath.ToSlash(path), Key: cleanKey,
		ContentType: contentType, SizeBytes: n,
	}, nil
}

func (s *FSStore) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	path := filepath.Join(s.root, cleanObjectKey(key))
	return "fs://" + filepath.ToSlash(path), nil
}

func cleanObjectKey(key string) string {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	key = strings.TrimPrefix(key, "/")
	if key == "" || strings.Contains(key, "..") {
		return "invalid-object-key"
	}
	return key
}

type UnsupportedStore struct {
	Kind   string
	Reason string
}

func (s UnsupportedStore) Put(context.Context, string, io.Reader, string) (*ObjectRef, error) {
	return nil, s.err()
}

func (s UnsupportedStore) SignedURL(context.Context, string, time.Duration) (string, error) {
	return "", s.err()
}

func (s UnsupportedStore) err() error {
	if s.Reason != "" {
		return fmt.Errorf("artifact store %q is not configured: %s", s.Kind, s.Reason)
	}
	return fmt.Errorf("artifact store %q is not configured", s.Kind)
}

type S3Config struct {
	Kind            string
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
}

type S3Store struct {
	cfg       S3Config
	client    *s3.Client
	presigner *s3.PresignClient
}

func NewS3Store(cfg S3Config) *S3Store {
	awsCfg := aws.Config{
		Region:      cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})
	return &S3Store{cfg: cfg, client: client, presigner: s3.NewPresignClient(client)}
}

func (s *S3Store) Put(ctx context.Context, key string, r io.Reader, contentType string) (*ObjectRef, error) {
	cleanKey := cleanObjectKey(key)
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(cleanKey),
		Body:   bytes.NewReader(body),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return nil, err
	}
	return &ObjectRef{
		URI:         "s3://" + s.cfg.Bucket + "/" + cleanKey,
		Key:         cleanKey,
		ContentType: contentType,
		SizeBytes:   int64(len(body)),
	}, nil
}

func (s *S3Store) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	res, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(cleanObjectKey(key)),
	}, func(o *s3.PresignOptions) {
		o.Expires = ttl
	})
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

func loadS3Config(kind string) (S3Config, Profile) {
	cfg := S3Config{
		Kind:            kind,
		Bucket:          strings.TrimSpace(os.Getenv("ASH_S3_BUCKET")),
		Region:          envOr("ASH_S3_REGION", "us-east-1"),
		Endpoint:        strings.TrimSpace(os.Getenv("ASH_S3_ENDPOINT")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("ASH_S3_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("ASH_S3_SECRET_ACCESS_KEY")),
		ForcePathStyle:  boolEnv("ASH_S3_FORCE_PATH_STYLE", kind == "s3-compatible"),
	}
	profile := Profile{
		Kind:              kind,
		ObjectStore:       true,
		SupportsSignedURL: true,
	}
	missing := missingS3Config(cfg)
	if len(missing) > 0 {
		profile.Error = "missing " + strings.Join(missing, ", ")
		return cfg, profile
	}
	profile.Ready = true
	profile.URI = "s3://" + cfg.Bucket
	return cfg, profile
}

func missingS3Config(cfg S3Config) []string {
	var missing []string
	if cfg.Bucket == "" {
		missing = append(missing, "ASH_S3_BUCKET")
	}
	if cfg.Region == "" {
		missing = append(missing, "ASH_S3_REGION")
	}
	if cfg.AccessKeyID == "" {
		missing = append(missing, "ASH_S3_ACCESS_KEY_ID")
	}
	if cfg.SecretAccessKey == "" {
		missing = append(missing, "ASH_S3_SECRET_ACCESS_KEY")
	}
	if cfg.Kind == "s3-compatible" && cfg.Endpoint == "" {
		missing = append(missing, "ASH_S3_ENDPOINT")
	}
	return missing
}

func envOr(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func boolEnv(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
