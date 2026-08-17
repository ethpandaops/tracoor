package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/mime"
	"github.com/sirupsen/logrus"
)

// uploadPartSize and uploadConcurrency bound how much of a stream is held in
// memory while it is being uploaded. The uploader buffers at most one part per
// worker plus one, so the pair is the real per-upload memory ceiling: the point
// of streaming is lost if the uploader buffers the payload on our behalf.
const (
	uploadPartSize    = manager.MinUploadPartSize
	uploadConcurrency = 4
)

const (
	// s3DeleteBatchSize is the hard limit DeleteObjects imposes on keys per request.
	s3DeleteBatchSize = 1000
	// s3NoSuchKeyCode is the per-key error code for an object that is already gone.
	s3NoSuchKeyCode = "NoSuchKey"
)

type S3Store struct {
	s3Client *s3.Client

	// uploader streams bodies of unknown length. PutObject cannot: it needs a
	// known content length, which a pipe does not have.
	uploader *manager.Uploader

	config *S3StoreConfig

	log  logrus.FieldLogger
	opts *Options

	basicMetrics *BasicMetrics
}

//nolint:tagliatelle // required snake.
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

	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = uploadPartSize
		u.Concurrency = uploadConcurrency
		// A streamed body cannot be hashed before it is sent, so the payload
		// signature is skipped exactly as it was on the buffered path.
		u.ClientOptions = append(
			u.ClientOptions,
			s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware),
		)
	})

	return &S3Store{
		s3Client:     s3Client,
		uploader:     uploader,
		config:       config,
		log:          log,
		opts:         opts,
		basicMetrics: metrics,
	}, nil
}

// countingReader records how many bytes were read out of a stream, so an upload
// of unknown length can still report its size once it has finished.
type countingReader struct {
	reader io.Reader
	read   int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.read += int64(n)

	return n, err
}

// putStream uploads params.Data to params.Location.
//
// A read failure part way through the body surfaces from Upload and the
// multipart upload is aborted rather than completed, so a truncated payload is
// never published under a location the index will later point at.
func (s *S3Store) putStream(ctx context.Context, params *SaveParams, dataType DataType, failure string) (string, error) {
	if params == nil || params.Data == nil {
		return "", errors.New("data is nil")
	}

	counter := &countingReader{reader: params.Data}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(params.Location),
		Body:   counter,
	}

	if params.ContentEncoding != "" {
		input.ContentEncoding = aws.String(params.ContentEncoding)
	}

	if _, err := s.uploader.Upload(ctx, input); err != nil {
		var apiErr smithy.APIError

		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *s3types.NoSuchBucket:
				return "", errors.New("bucket does not exist: " + apiErr.Error())
			case *s3types.NotFound:
				return "", ErrNotFound
			default:
				return "", errors.New(failure + ": " + apiErr.Error())
			}
		}

		return "", fmt.Errorf("%s: %w", failure, err)
	}

	s.basicMetrics.ObserveItemAdded(string(dataType))
	s.basicMetrics.ObserveItemAddedBytes(string(dataType), int(counter.read))

	return params.Location, nil
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
	return s.putStream(ctx, params, BeaconStateDataType, "failed to save beacon state")
}

func (s *S3Store) getPresignedURL(ctx context.Context, params *GetURLParams) (string, error) {
	presignClient := s3.NewPresignClient(s.s3Client)

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
	return s.putStream(ctx, params, BeaconBlockDataType, "failed to save beacon block")
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

func (s *S3Store) SaveExecutionPayloadEnvelope(ctx context.Context, params *SaveParams) (string, error) {
	return s.putStream(ctx, params, ExecutionPayloadEnvelopeDataType, "failed to save execution payload envelope")
}

func (s *S3Store) GetExecutionPayloadEnvelopeURL(ctx context.Context, params *GetURLParams) (string, error) {
	url, err := s.getPresignedURL(ctx, params)
	if err != nil {
		return "", err
	}

	s.basicMetrics.ObserveItemURLRetreived(string(ExecutionPayloadEnvelopeDataType))

	return url, nil
}

func (s *S3Store) GetExecutionPayloadEnvelope(ctx context.Context, location string) (*[]byte, error) {
	s.basicMetrics.ObserveCacheMiss(string(ExecutionPayloadEnvelopeDataType))

	data, err := s.GetRaw(ctx, location)
	if err != nil {
		return nil, err
	}

	s.basicMetrics.ObserveItemRetreived(string(ExecutionPayloadEnvelopeDataType))

	b := data.Bytes()

	return &b, nil
}

func (s *S3Store) DeleteExecutionPayloadEnvelope(ctx context.Context, location string) error {
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

	s.basicMetrics.ObserveItemRemoved(string(ExecutionPayloadEnvelopeDataType))

	return err
}

func (s *S3Store) SaveBeaconBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	return s.putStream(ctx, params, BeaconBadBlockDataType, "failed to save beacon bad block")
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
	return s.putStream(ctx, params, BeaconBadBlobDataType, "failed to save beacon bad blob")
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
	return s.putStream(ctx, params, BlockTraceDataType, "failed to save execution block trace")
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
	return s.putStream(ctx, params, BadBlockDataType, "failed to save execution bad block")
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

func (s *S3Store) Copy(ctx context.Context, params *CopyParams) error {
	if params.Source == "" || params.Destination == "" {
		return errors.New("source and destination are required")
	}

	// Log the copy operation attempt
	s.log.WithFields(logrus.Fields{
		"source":      params.Source,
		"destination": params.Destination,
	}).Debug("Performing object copy")

	// First try to do a regular server-side copy (works with MinIO and S3)
	_, err := s.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.config.BucketName),
		CopySource: aws.String(fmt.Sprintf("%s/%s", s.config.BucketName, params.Source)),
		Key:        aws.String(params.Destination),
	})

	// If the server-side copy was successful, return
	if err == nil {
		return nil
	}

	// If we got an error, check if it's a specific R2 error or a general error
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		// For R2, we need to use GetObject + PutObject as a workaround
		// But we only do this for specific errors that indicate a compatibility issue
		// Log that we're falling back to the manual copy method
		s.log.WithFields(logrus.Fields{
			"source":      params.Source,
			"destination": params.Destination,
			"error":       apiErr.Error(),
			"errorCode":   apiErr.ErrorCode(),
		}).Debug("Server-side copy failed, falling back to GetObject + PutObject method")

		// Get the source object
		getResult, getErr := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.config.BucketName),
			Key:    aws.String(params.Source),
		})
		if getErr != nil {
			return fmt.Errorf("failed to get source object for manual copy: %w", getErr)
		}
		defer getResult.Body.Close()

		// Read the source object content
		buf := new(bytes.Buffer)
		if _, readErr := buf.ReadFrom(getResult.Body); readErr != nil {
			return fmt.Errorf("failed to read source object content: %w", readErr)
		}

		// Create the put input with the same metadata as the source
		putInput := &s3.PutObjectInput{
			Bucket:             aws.String(s.config.BucketName),
			Key:                aws.String(params.Destination),
			Body:               bytes.NewReader(buf.Bytes()),
			ContentType:        getResult.ContentType,
			ContentDisposition: getResult.ContentDisposition,
			ContentEncoding:    getResult.ContentEncoding,
			ContentLanguage:    getResult.ContentLanguage,
			CacheControl:       getResult.CacheControl,
			Expires:            getResult.Expires,
		}

		// Put the object
		_, putErr := s.s3Client.PutObject(ctx, putInput)
		if putErr != nil {
			return fmt.Errorf("failed to put object in manual copy: %w", putErr)
		}

		return nil
	}

	// If not an API error, return the original error
	return fmt.Errorf("failed to copy object: %w", err)
}

// DeleteMany removes objects in bulk. Keys are sent in DeleteObjects batches; a batch that
// fails wholesale and individual per-key errors both surface as failed locations rather than
// aborting the remaining batches, because retention deletes a mixed bag of objects and one
// poisoned key must not keep the rest alive forever.
func (s *S3Store) DeleteMany(ctx context.Context, locations []string) error {
	if len(locations) == 0 {
		return nil
	}

	var (
		failed   []string
		firstErr error
	)

	for start := 0; start < len(locations); start += s3DeleteBatchSize {
		end := start + s3DeleteBatchSize
		if end > len(locations) {
			end = len(locations)
		}

		chunk := locations[start:end]

		objects := make([]s3types.ObjectIdentifier, 0, len(chunk))
		for _, location := range chunk {
			objects = append(objects, s3types.ObjectIdentifier{Key: aws.String(location)})
		}

		out, err := s.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.config.BucketName),
			Delete: &s3types.Delete{
				Objects: objects,
				// Quiet still reports errors, it only drops the per-key success entries.
				Quiet: aws.Bool(true),
			},
		})
		if err != nil {
			failed = append(failed, chunk...)

			if firstErr == nil {
				firstErr = err
			}

			continue
		}

		removed := len(chunk)

		for _, e := range out.Errors {
			key := aws.ToString(e.Key)

			// A key that is already gone is the outcome we wanted.
			if aws.ToString(e.Code) == s3NoSuchKeyCode {
				continue
			}

			removed--

			failed = append(failed, key)

			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %s", key, aws.ToString(e.Message))
			}
		}

		for i := 0; i < removed; i++ {
			s.basicMetrics.ObserveItemRemoved(string(UnknownDataType))
		}
	}

	if len(failed) > 0 {
		return &DeleteManyError{Failed: failed, Err: firstErr}
	}

	return nil
}

func (s *S3Store) PreferURLs() bool {
	return s.config.PreferURLs
}
