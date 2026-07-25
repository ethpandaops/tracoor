package store

import (
	"context"
	"errors"
	"fmt"
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

// resolve turns a location into an absolute path rooted at the store's base
// path, and rejects it if the resolved path would fall outside that base
// path (for example because location contains ".." segments or is an
// absolute path of its own). Every method that turns a caller-supplied
// location into a filesystem path must go through this, since location
// values are not otherwise validated before reaching the store.
func (s *FSStore) resolve(location string) (string, error) {
	parts := strings.Split(location, "/")
	joined := filepath.Join(s.basePath, filepath.Join(parts...))

	base, err := filepath.Abs(s.basePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}

	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("failed to resolve location: %w", err)
	}

	if resolved != base && !strings.HasPrefix(resolved, base+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: location %q resolves outside the store's base path", ErrInvalid, location)
	}

	return resolved, nil
}

func (s *FSStore) Exists(ctx context.Context, location string) (bool, error) {
	path, err := s.resolve(location)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
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

func (s *FSStore) saveFile(data *[]byte, path string) error {
	if err := s.ensureDir(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, *data, 0o600); err != nil {
		return err
	}

	return nil
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

func (s *FSStore) SaveBeaconState(ctx context.Context, params *SaveParams) (string, error) {
	path, err := s.resolve(params.Location)
	if err != nil {
		return "", err
	}

	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconState(ctx context.Context, location string) (*[]byte, error) {
	path, err := s.resolve(location)
	if err != nil {
		return nil, err
	}

	return s.getFile(path)
}

func (s *FSStore) GetBeaconStateURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconState(ctx context.Context, location string) error {
	path, err := s.resolve(location)
	if err != nil {
		return err
	}

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBlock(ctx context.Context, params *SaveParams) (string, error) {
	path, err := s.resolve(params.Location)
	if err != nil {
		return "", err
	}

	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconBlock(ctx context.Context, location string) (*[]byte, error) {
	path, err := s.resolve(location)
	if err != nil {
		return nil, err
	}

	return s.getFile(path)
}

func (s *FSStore) GetBeaconBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBlock(ctx context.Context, location string) error {
	path, err := s.resolve(location)
	if err != nil {
		return err
	}

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	path, err := s.resolve(params.Location)
	if err != nil {
		return "", err
	}

	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconBadBlock(ctx context.Context, location string) (*[]byte, error) {
	path, err := s.resolve(location)
	if err != nil {
		return nil, err
	}

	return s.getFile(path)
}

func (s *FSStore) GetBeaconBadBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBadBlock(ctx context.Context, location string) error {
	path, err := s.resolve(location)
	if err != nil {
		return err
	}

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBadBlob(ctx context.Context, params *SaveParams) (string, error) {
	path, err := s.resolve(params.Location)
	if err != nil {
		return "", err
	}

	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetBeaconBadBlob(ctx context.Context, location string) (*[]byte, error) {
	path, err := s.resolve(location)
	if err != nil {
		return nil, err
	}

	return s.getFile(path)
}

func (s *FSStore) GetBeaconBadBlobURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBadBlob(ctx context.Context, location string) error {
	path, err := s.resolve(location)
	if err != nil {
		return err
	}

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionBlockTrace(ctx context.Context, params *SaveParams) (string, error) {
	path, err := s.resolve(params.Location)
	if err != nil {
		return "", err
	}

	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetExecutionBlockTrace(ctx context.Context, location string) (*[]byte, error) {
	path, err := s.resolve(location)
	if err != nil {
		return nil, err
	}

	return s.getFile(path)
}

func (s *FSStore) GetExecutionBlockTraceURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionBlockTrace(ctx context.Context, location string) error {
	path, err := s.resolve(location)
	if err != nil {
		return err
	}

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionBadBlock(ctx context.Context, params *SaveParams) (string, error) {
	path, err := s.resolve(params.Location)
	if err != nil {
		return "", err
	}

	if err := s.saveFile(params.Data, path); err != nil {
		return "", err
	}

	return params.Location, nil
}

func (s *FSStore) GetExecutionBadBlock(ctx context.Context, location string) (*[]byte, error) {
	path, err := s.resolve(location)
	if err != nil {
		return nil, err
	}

	return s.getFile(path)
}

func (s *FSStore) GetExecutionBadBlockURL(ctx context.Context, params *GetURLParams) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionBadBlock(ctx context.Context, location string) error {
	path, err := s.resolve(location)
	if err != nil {
		return err
	}

	return s.removeFile(path)
}

func (s *FSStore) PathPrefix() string {
	return s.basePath
}

func (s *FSStore) PreferURLs() bool {
	return false
}

func (s *FSStore) StorageHandshakeTokenExists(ctx context.Context, node string) (bool, error) {
	return s.Exists(ctx, filepath.Join("handshake_tokens", node))
}

func (s *FSStore) SaveStorageHandshakeToken(ctx context.Context, node, data string) error {
	path, err := s.resolve(filepath.Join("handshake_tokens", node))
	if err != nil {
		return err
	}

	dataBytes := []byte(data)

	return s.saveFile(&dataBytes, path)
}

func (s *FSStore) GetStorageHandshakeToken(ctx context.Context, node string) (string, error) {
	path, err := s.resolve(filepath.Join("handshake_tokens", node))
	if err != nil {
		return "", err
	}

	data, err := s.getFile(path)
	if err != nil {
		return "", err
	}

	return string(*data), nil
}

func (s *FSStore) Copy(ctx context.Context, params *CopyParams) error {
	if params.Source == "" || params.Destination == "" {
		return errors.New("source and destination are required")
	}

	source, err := s.resolve(params.Source)
	if err != nil {
		return err
	}

	destination, err := s.resolve(params.Destination)
	if err != nil {
		return err
	}

	if err := s.ensureDir(destination); err != nil {
		return err
	}

	// Read the source file
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to the destination file
	if err := os.WriteFile(destination, data, 0o600); err != nil { //nolint:gosec // destination is validated by resolve()
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
