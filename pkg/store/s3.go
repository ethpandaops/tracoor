package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/mime"
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
	PreferURLs   bool   `yaml:"prefer_urls"`
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

	metrics := GetBasicMetricsInstance(namespace, string(S3StoreType), opts.MetricsEnabled)

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

func (s *S3Store) SaveBeaconState(ctx context.Context, params *SaveParams) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   bytes.NewBuffer(*params.Data),
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	_, err := s.s3Client.PutObject(ctx, input, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	if err != nil {
		var apiErr smithy.APIError

		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NoSuchBucket:
				return "", errors.New("bucket does not exist: " + apiErr.Error())
			case *s3types.NotFound:
				return "", ErrNotFound
			default:
				return "", errors.New("failed to save beacon state: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemAdded(string(BeaconStateDataType))
	s.basicMetrics.ObserveItemAddedBytes(string(BeaconStateDataType), len(*params.Data))

	return params.Location, err
}

func (s *S3Store) getPresignedURL(ctx context.Context, params *GetURLParams) (string, error) {
	presignClient := s3.NewPresignClient(s.s3Client)

	s.log.WithFields(logrus.Fields{
		"location":         params.Location,
		"expiry":           params.Expiry,
		"content_encoding": params.ContentEncoding,
	}).Info("Getting presigned URL")

	// Remove the compression extension if it exists
	extension := filepath.Ext(
		compression.RemoveExtension(
			params.Location,
		),
	)

	input := &s3.GetObjectInput{
		Bucket:              aws.String(s.config.BucketName),
		Key:                 aws.String(params.Location),
		ResponseContentType: aws.String(string(mime.GetContentTypeFromExtension(extension))),
		ResponseContentDisposition: aws.String(
			fmt.Sprintf("attachment; filename=%q", compression.RemoveExtension(
				filepath.Base(params.Location),
			)),
		),
	}

	// Backwards compatibility for old locations that still have the compression algorithm in the filename
	if params.ContentEncoding == "" {
		compressionAlgorithm, err := compression.GetCompressionAlgorithm(params.Location)
		if err == nil {
			// Set the content encoding
			input.ResponseContentEncoding = aws.String(compressionAlgorithm.ContentEncoding)

			extension = compression.RemoveExtension(extension)

			// Set the content type correctly. Without this, a filename of data.json.gz would be detected as a .gz rather than a .json
			input.ResponseContentDisposition = aws.String(
				fmt.Sprintf("attachment; filename=%q",
					compression.RemoveExtension(
						filepath.Base(params.Location),
					),
				),
			)
			input.ResponseContentType = aws.String(string(
				mime.GetContentTypeFromExtension(extension),
			))
		}
	} else {
		input.ResponseContentEncoding = aws.String(params.ContentEncoding)
	}

	s.log.WithFields(logrus.Fields{
		"extension":           extension,
		"content_disposition": input.ResponseContentDisposition,
		"content_type":        input.ResponseContentType,
		"content_encoding":    input.ResponseContentEncoding,
		"location":            params.Location,
		"expiry":              params.Expiry,
	}).Info("Generated presigned URL")

	resp, err := presignClient.PresignGetObject(ctx, input, s3.WithPresignExpires(time.Duration(params.Expiry)*time.Second))
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

func (s *S3Store) GetBeaconStateURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(BeaconStateDataType))

	return url, nil
}

func (s *S3Store) GetBeaconState(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BeaconStateDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(BeaconStateDataType))

	b := data.Bytes()

	return &b, nil
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

func (s *S3Store) SaveBeaconBlock(ctx context.Context, params *SaveParams) (string, error) {
	if params.Data == nil {
		return "", errors.New("data is nil")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   bytes.NewBuffer(*params.Data),
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	_, err := s.s3Client.PutObject(ctx, input, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
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

	s.basicMetrics.ObserveItemAdded(string(BeaconBlockDataType))
	s.basicMetrics.ObserveItemAddedBytes(string(BeaconBlockDataType), len(*params.Data))

	return params.Location, err
}

func (s *S3Store) GetBeaconBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(BeaconBlockDataType))

	return url, nil
}

func (s *S3Store) GetBeaconBlock(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BeaconBlockDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(BeaconBlockDataType))

	b := data.Bytes()

	return &b, nil
}

func (s *S3Store) DeleteBeaconBlock(ctx context.Context, location string) error {
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

	s.basicMetrics.ObserveItemRemoved(string(BeaconBlockDataType))

	return err
}

func (s *S3Store) SaveBeaconBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	if params.Data == nil {
		return "", errors.New("data is nil")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   bytes.NewBuffer(*params.Data),
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	_, err := s.s3Client.PutObject(ctx, input, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
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

	s.basicMetrics.ObserveItemAdded(string(BeaconBadBlockDataType))
	s.basicMetrics.ObserveItemAddedBytes(string(BeaconBadBlockDataType), len(*params.Data))

	return params.Location, err
}

func (s *S3Store) GetBeaconBadBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(BeaconBadBlockDataType))

	return url, nil
}

func (s *S3Store) GetBeaconBadBlock(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BeaconBadBlockDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(BeaconBadBlockDataType))

	b := data.Bytes()

	return &b, nil
}

func (s *S3Store) DeleteBeaconBadBlock(ctx context.Context, location string) error {
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

	s.basicMetrics.ObserveItemRemoved(string(BeaconBadBlockDataType))

	return err
}

func (s *S3Store) SaveBeaconBadBlob(ctx context.Context, params *SaveParams) (string, error) {
	if params.Data == nil {
		return "", errors.New("data is nil")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   bytes.NewBuffer(*params.Data),
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	_, err := s.s3Client.PutObject(ctx, input, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
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

	s.basicMetrics.ObserveItemAdded(string(BeaconBadBlobDataType))
	s.basicMetrics.ObserveItemAddedBytes(string(BeaconBadBlobDataType), len(*params.Data))

	return params.Location, err
}

func (s *S3Store) GetBeaconBadBlobURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(BeaconBadBlobDataType))

	return url, nil
}

func (s *S3Store) GetBeaconBadBlob(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BeaconBadBlobDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(BeaconBadBlobDataType))

	b := data.Bytes()

	return &b, nil
}

func (s *S3Store) DeleteBeaconBadBlob(ctx context.Context, location string) error {
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

	s.basicMetrics.ObserveItemRemoved(string(BeaconBadBlobDataType))

	return err
}

func (s *S3Store) SaveExecutionBlockTrace(ctx context.Context, params *SaveParams) (string, error) {
	if params.Data == nil {
		return "", errors.New("data is nil")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   bytes.NewBuffer(*params.Data),
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	_, err := s.s3Client.PutObject(ctx, input, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	if err != nil {
		var apiErr smithy.APIError

		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NoSuchBucket:
				return "", errors.New("bucket does not exist: " + apiErr.Error())
			case *s3types.NotFound:
				return "", ErrNotFound
			default:
				return "", errors.New("failed to save execution block trace: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemAdded(string(BlockTraceDataType))
	s.basicMetrics.ObserveItemAddedBytes(string(BlockTraceDataType), len(*params.Data))

	return params.Location, err
}

func (s *S3Store) GetExecutionBlockTrace(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BlockTraceDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(BlockTraceDataType))

	b := data.Bytes()

	return &b, nil

}

func (s *S3Store) GetExecutionBlockTraceURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(BlockTraceDataType))

	return url, nil
}

func (s *S3Store) DeleteExecutionBlockTrace(ctx context.Context, location string) error {
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
				return errors.New("failed to delete execution block trace: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemRemoved(string(BlockTraceDataType))

	return err
}

func (s *S3Store) SaveExecutionBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	if params.Data == nil {
		return "", errors.New("data is nil")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   bytes.NewBuffer(*params.Data),
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	_, err := s.s3Client.PutObject(ctx, input, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	if err != nil {
		var apiErr smithy.APIError

		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NoSuchBucket:
				return "", errors.New("bucket does not exist: " + apiErr.Error())
			case *s3types.NotFound:
				return "", ErrNotFound
			default:
				return "", errors.New("failed to save execution block trace: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemAdded(string(BadBlockDataType))
	s.basicMetrics.ObserveItemAddedBytes(string(BadBlockDataType), len(*params.Data))

	return params.Location, err
}

func (s *S3Store) GetExecutionBadBlock(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(BadBlockDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(BadBlockDataType))

	b := data.Bytes()

	return &b, nil
}

func (s *S3Store) GetExecutionBadBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(BadBlockDataType))

	return url, nil
}

func (s *S3Store) DeleteExecutionBadBlock(ctx context.Context, location string) error {
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
				return errors.New("failed to delete execution block trace: " + apiErr.Error())
			}
		}
	}

	s.basicMetrics.ObserveItemRemoved(string(BadBlockDataType))

	return err
}

func (s *S3Store) PreferURLs() bool {
	return s.config.PreferURLs
}
