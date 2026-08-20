package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

//nolint:tagliatelle // requires snake.
type FSStoreConfig struct {
	BasePath string `yaml:"base_path"`
}

type FSStore struct {
	basePath string
	log      logrus.FieldLogger
}

func NewFSStore(namespace string, log logrus.FieldLogger, config *FSStoreConfig, opts *Options) (*FSStore, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if config.BasePath == "" {
		return nil, errors.New("base path is required")
	}

	// Ensure the base path exists, create it if necessary
	if err := os.MkdirAll(config.BasePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &FSStore{
		basePath: config.BasePath,
		log:      log,
	}, nil
}

func (s *FSStore) Healthy(ctx context.Context) error {
	if _, err := os.Stat(s.basePath); os.IsNotExist(err) {
		return fmt.Errorf("base path does not exist: %s", s.basePath)
	}

	return nil
}

func (s *FSStore) Exists(ctx context.Context, location string) (bool, error) {
	parts := strings.Split(location, "/")

	_, err := os.Stat(filepath.Join(s.basePath, filepath.Join(parts...)))
	if os.IsNotExist(err) {
		return false, nil
	}

	return err == nil, err
}

func (s *FSStore) ensureDir(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return nil
}

// saveFile streams data into a temporary file alongside path and renames it
// into place once the whole body has been written. Publishing is therefore
// atomic: a reader that fails part way through leaves the temporary file to be
// removed and nothing at path, so no other process can observe a truncated
// object as a complete one.
func (s *FSStore) saveFile(data io.Reader, path string) error {
	if data == nil {
		return errors.New("data is nil")
	}

	if err := s.ensureDir(path); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tmpPath := tmp.Name()

	// Every exit short of a completed rename has to take the temporary file
	// with it, or a failed save leaves litter beside the real object.
	renamed := false

	defer func() {
		if renamed {
			return
		}

		_ = tmp.Close()

		if rerr := os.Remove(tmpPath); rerr != nil && !os.IsNotExist(rerr) {
			s.log.WithError(rerr).WithField("path", tmpPath).Warn("Failed to remove temporary file")
		}
	}()

	if _, err := io.Copy(tmp, data); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to publish %s: %w", path, err)
	}

	renamed = true

	return nil
}

func (s *FSStore) constructLocation(parts ...string) string {
	return filepath.Join(parts...)
}

func (s *FSStore) getFile(path string) (*[]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *FSStore) removeFile(path string) error {
	return os.Remove(path)
}

func (s *FSStore) SaveRaw(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) SaveBeaconState(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconState(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetBeaconStateURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconState(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBlock(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconBlock(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetBeaconBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBlock(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionPayloadEnvelope(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetExecutionPayloadEnvelope(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetExecutionPayloadEnvelopeURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionPayloadEnvelope(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconBadBlock(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.getFile(path)
}

func (s *FSStore) GetBeaconBadBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBadBlock(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBadBlob(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconBadBlob(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetBeaconBadBlobURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBadBlob(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionBlockTrace(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetExecutionBlockTrace(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetExecutionBlockTraceURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionBlockTrace(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	parts := strings.Split(params.Location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetExecutionBadBlock(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetExecutionBadBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionBadBlock(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")

	return s.removeFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

// DeleteMany removes objects in bulk. There is no batch primitive on a filesystem, so this is
// a loop that keeps going past individual failures and reports the ones that did not go, in
// the same shape as the object-store implementation.
func (s *FSStore) DeleteMany(ctx context.Context, locations []string) error {
	var (
		failed   []string
		firstErr error
	)

	for idx, location := range locations {
		if err := ctx.Err(); err != nil {
			// Everything not yet attempted is still there; say so rather than claiming success.
			failed = append(failed, locations[idx:]...)

			return &DeleteManyError{Failed: failed, Err: err}
		}

		parts := strings.Split(location, "/")

		if err := s.removeFile(filepath.Join(s.basePath, filepath.Join(parts...))); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			failed = append(failed, location)

			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if len(failed) > 0 {
		return &DeleteManyError{Failed: failed, Err: firstErr}
	}

	return nil
}

func (s *FSStore) PathPrefix() string {
	return s.basePath
}

func (s *FSStore) PreferURLs() bool {
	return false
}

func (s *FSStore) StorageHandshakeTokenExists(ctx context.Context, node string) (bool, error) {
	location := s.constructLocation("handshake_tokens", node)

	exists, err := s.Exists(ctx, location)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *FSStore) SaveStorageHandshakeToken(ctx context.Context, node, data string) error {
	location := s.constructLocation(s.basePath, "handshake_tokens", node)

	if err := s.saveFile(bytes.NewReader([]byte(data)), location); err != nil {
		return err
	}

	return nil
}

func (s *FSStore) GetStorageHandshakeToken(ctx context.Context, node string) (string, error) {
	location := s.constructLocation(s.basePath, "handshake_tokens", node)

	data, err := s.getFile(location)
	if err != nil {
		return "", err
	}

	return string(*data), nil
}

func (s *FSStore) Copy(ctx context.Context, params *CopyParams) error {
	if params.Source == "" || params.Destination == "" {
		return errors.New("source and destination are required")
	}

	source := filepath.Join(s.basePath, params.Source)
	destination := filepath.Join(s.basePath, params.Destination)

	from, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	defer func() {
		if cerr := from.Close(); cerr != nil {
			s.log.WithError(cerr).WithField("path", source).Warn("Failed to close source file")
		}
	}()

	// Streamed and published by rename, for the same reason a save is: a payload can be tens
	// of megabytes, and a reader must never find a half-written object at the destination.
	if err := s.saveFile(from, destination); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
