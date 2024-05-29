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

func (s *FSStore) saveFile(data *[]byte, path string) error {
	if err := s.ensureDir(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, *data, 0o600); err != nil {
		return err
	}

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

func (s *FSStore) SaveBeaconState(ctx context.Context, data *[]byte, location string) (string, error) {
	parts := strings.Split(location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(data, path); err != nil {
		return "", err
	}

	return location, nil
}

func (s *FSStore) GetBeaconState(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetBeaconStateURL(ctx context.Context, location string, expiry int) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconState(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBlock(ctx context.Context, data *[]byte, location string) (string, error) {
	parts := strings.Split(location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(data, path); err != nil {
		return "", err
	}

	return location, nil
}

func (s *FSStore) GetBeaconBlock(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetBeaconBlockURL(ctx context.Context, location string, expiry int) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBlock(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBadBlock(ctx context.Context, data *[]byte, location string) (string, error) {
	parts := strings.Split(location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(data, path); err != nil {
		return "", err
	}

	return location, nil
}

func (s *FSStore) GetBeaconBadBlock(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.getFile(path)
}

func (s *FSStore) GetBeaconBadBlockURL(ctx context.Context, location string, expiry int) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBadBlock(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveBeaconBadBlob(ctx context.Context, data *[]byte, location string) (string, error) {
	parts := strings.Split(location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(data, path); err != nil {
		return "", err
	}

	return location, nil
}

func (s *FSStore) GetBeaconBadBlob(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetBeaconBadBlobURL(ctx context.Context, location string, expiry int) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteBeaconBadBlob(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionBlockTrace(ctx context.Context, data *[]byte, location string) (string, error) {
	parts := strings.Split(location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(data, path); err != nil {
		return "", err
	}

	return location, nil
}

func (s *FSStore) GetExecutionBlockTrace(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetExecutionBlockTraceURL(ctx context.Context, location string, expiry int) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionBlockTrace(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")
	path := filepath.Join(s.basePath, filepath.Join(parts...))

	return s.removeFile(path)
}

func (s *FSStore) SaveExecutionBadBlock(ctx context.Context, data *[]byte, location string) (string, error) {
	parts := strings.Split(location, "/")

	path := filepath.Join(s.basePath, filepath.Join(parts...))
	if err := s.saveFile(data, path); err != nil {
		return "", err
	}

	return location, nil
}

func (s *FSStore) GetExecutionBadBlock(ctx context.Context, location string) (*[]byte, error) {
	parts := strings.Split(location, "/")

	return s.getFile(filepath.Join(s.basePath, filepath.Join(parts...)))
}

func (s *FSStore) GetExecutionBadBlockURL(ctx context.Context, location string, expiry int) (string, error) {
	return "", errors.New("not supported")
}

func (s *FSStore) DeleteExecutionBadBlock(ctx context.Context, location string) error {
	parts := strings.Split(location, "/")

	return s.removeFile(filepath.Join(s.basePath, filepath.Join(parts...)))
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
	dataBytes := []byte(data)

	if err := s.saveFile(&dataBytes, location); err != nil {
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
