package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/sirupsen/logrus"
)

type S3Store struct {
	s3Client *s3.Client

	config *S3StoreConfig

	log  logrus.FieldLogger
	opts *Options

	basicMetrics *BasicMetrics
}

type S3StoreConfig struct {
	Endpoint     string `yaml:"endpoint"`
	Region       string `yaml:"region"`
	BucketName   string `yaml:"bucket_name"`
	KeyPrefix    string `yaml:"key_prefix"`
	AccessKey    string `yaml:"access_key"`
	AccessSecret string `yaml:"access_secret"`
	UsePathStyle bool   `yaml:"use_path_style"`
}

// NewS3Store creates a new S3Store instance with the specified AWS configuration, bucket name, and key prefix.
func NewS3Store(namespace string, log logrus.FieldLogger, config *S3StoreConfig, opts *Options) (*S3Store, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:       "aws",
			SigningRegion:     config.Region,
			URL:               config.Endpoint,
			HostnameImmutable: true,
		}, nil
	})

	cfg := aws.Config{
		Region:                      config.Region,
		EndpointResolverWithOptions: resolver,
		Credentials:                 credentials.NewStaticCredentialsProvider(config.AccessKey, config.AccessSecret, ""),
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = config.UsePathStyle
	})

	metrics := NewBasicMetrics(namespace, string(S3StoreType), opts.MetricsEnabled)

	return &S3Store{
		s3Client:     s3Client,
		config:       config,
		log:          log,
		opts:         opts,
		basicMetrics: metrics,
	}, nil
}

func (s *S3Store) PathPrefix() string {
	return s.config.KeyPrefix
}

func (s *S3Store) Healthy(ctx context.Context) error {
	_, err := s.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Store) SaveBeaconState(ctx context.Context, data *[]byte, location string) (string, error) {
	compressed, err := GzipCompress(*data)
	if err != nil {
		return "", err
	}

	location = location + ".gz"

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(location),
		Body:   bytes.NewBuffer(compressed),
	}, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NoSuchBucket:
				return "", errors.New("bucket does not exist: " + apiErr.Error())
			case *s3types.NotFound:
				return "", ErrNotFound
			default:
				return "", errors.New("failed to save frame: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemAdded(string(BeaconStateDataType))

	return location, err
}

func (s *S3Store) GetRaw(ctx context.Context, location string) (*bytes.Buffer, error) {
	data, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(location),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NotFound:
				return nil, ErrNotFound
			default:
				return nil, errors.New("failed to get: " + apiErr.Error())
			}
		}

		return nil, err
	}
	defer data.Body.Close()

	var buff bytes.Buffer

	_, err = buff.ReadFrom(data.Body)
	if err != nil {
		return nil, err
	}

	return &buff, nil
}

func (s *S3Store) StorageHandshakeTokenExists(ctx context.Context, node string) (bool, error) {
	key := fmt.Sprintf("handshake/%s", node)
	_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NotFound:
				return false, nil
			default:
				return false, errors.New("failed to check if storage handshake token exists: " + apiErr.Error())
			}
		}

		return false, err
	}

	return true, nil
}

func (s *S3Store) SaveStorageHandshakeToken(ctx context.Context, node, data string) error {
	key := fmt.Sprintf("handshake/%s", node)

	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(key),
		Body:   strings.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to save storage handshake for node %s: %w", node, err)
	}

	return nil
}

func (s *S3Store) GetStorageHandshakeToken(ctx context.Context, node string) (string, error) {
	key := fmt.Sprintf("handshake/%s", node)
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NotFound:
				return "", ErrNotFound
			default:
				return "", errors.New("failed to get: " + apiErr.Error())
			}
		}

		return "", fmt.Errorf("failed to get storage handshake for node %s: %w", node, err)
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read storage handshake for node %s: %w", node, err)
	}

	return buf.String(), nil
}

func (s *S3Store) GetBeaconState(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BeaconStateDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(".gz", location) {
		b := data.Bytes()

		return &b, nil
	}

	s.basicMetrics.ObserveItemRetreived(string(BeaconStateDataType))

	uncompressed, err := GzipDecompress(data.Bytes())
	if err != nil {
		return nil, err
	}

	return &uncompressed, nil
}

func (s *S3Store) DeleteBeaconState(ctx context.Context, location string) error {
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(location),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NotFound:
				return ErrNotFound
			default:
				return errors.New("failed to delete: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemRemoved(string(BeaconStateDataType))

	return err
}

func (s *S3Store) Exists(ctx context.Context, location string) (bool, error) {
	_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(location),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NotFound" {
				return false, nil
			}
		}

		return false, err
	}

	return true, nil
}
