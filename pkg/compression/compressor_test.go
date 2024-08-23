package compression_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/tracoor/pkg/compression"
)

func TestNewCompressor(t *testing.T) {
	c := compression.NewCompressor()
	assert.NotNil(t, c)
}

func TestCompressor_Compress(t *testing.T) {
	c := compression.NewCompressor()

	testCases := []struct {
		name      string
		data      []byte
		algorithm *compression.CompressionAlgorithm
		wantErr   bool
	}{
		{
			name:      "Compress with Gzip",
			data:      []byte("test data"),
			algorithm: compression.Gzip,
			wantErr:   false,
		},
		{
			name:      "Compress with unsupported algorithm",
			data:      []byte("test data"),
			algorithm: &compression.CompressionAlgorithm{Name: "unsupported"},
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			compressed, err := c.Compress(&testCase.data, testCase.algorithm)
			if testCase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, compressed)
				assert.NotEqual(t, testCase.data, compressed)
			}
		})
	}
}

func TestCompressor_Decompress(t *testing.T) {
	c := compression.NewCompressor()

	testData := []byte("test data")
	compressed, err := c.Compress(&testData, compression.Gzip)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		data     []byte
		filename string
		wantErr  bool
	}{
		{
			name:     "Decompress Gzip",
			data:     compressed,
			filename: "test.gz",
			wantErr:  false,
		},
		{
			name:     "Decompress with nil data",
			data:     nil,
			filename: "test.gz",
			wantErr:  true,
		},
		{
			name:     "Decompress with unsupported algorithm",
			data:     []byte("test data"),
			filename: "test.unsupported",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			decompressed, err := c.Decompress(&testCase.data, testCase.filename)
			if testCase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testData, decompressed)
			}
		})
	}
}

func TestAddExtension(t *testing.T) {
	testCases := []struct {
		name      string
		filename  string
		algorithm *compression.CompressionAlgorithm
		want      string
	}{
		{
			name:      "Add Gzip extension",
			filename:  "test",
			algorithm: compression.Gzip,
			want:      "test.gz",
		},
		{
			name:      "Extension already present",
			filename:  "test.gz",
			algorithm: compression.Gzip,
			want:      "test.gz",
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			result := compression.AddExtension(testCase.filename, testCase.algorithm)
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestRemoveExtension(t *testing.T) {
	testCases := []struct {
		name      string
		filename  string
		algorithm *compression.CompressionAlgorithm
		want      string
	}{
		{
			name:      "Remove Gzip extension",
			filename:  "test.gz",
			algorithm: compression.Gzip,
			want:      "test",
		},
		{
			name:      "No extension to remove",
			filename:  "test",
			algorithm: compression.Gzip,
			want:      "test",
		},
		{
			name:      "Remove Gzip extension from .json.gz",
			filename:  "test.json.gz",
			algorithm: compression.Gzip,
			want:      "test.json",
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			result := compression.RemoveExtension(testCase.filename)
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestHasCompressionExtension(t *testing.T) {
	testCases := []struct {
		name      string
		filename  string
		algorithm *compression.CompressionAlgorithm
		want      bool
	}{
		{
			name:      "Has Gzip extension",
			filename:  "test.gz",
			algorithm: compression.Gzip,
			want:      true,
		},
		{
			name:      "No Gzip extension",
			filename:  "test",
			algorithm: compression.Gzip,
			want:      false,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			result := compression.HasCompressionExtension(testCase.filename, testCase.algorithm)
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestGetCompressionAlgorithm(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		want     *compression.CompressionAlgorithm
		wantErr  bool
	}{
		{
			name:     "Get Gzip algorithm",
			filename: "test.gz",
			want:     compression.Gzip,
			wantErr:  false,
		},
		{
			name:     "Unsupported algorithm",
			filename: "test.unsupported",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			result, err := compression.GetCompressionAlgorithm(testCase.filename)
			if testCase.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.want, result)
			}
		})
	}
}

func TestCompressAndDecompress(t *testing.T) {
	c := compression.NewCompressor()

	testCases := []struct {
		name      string
		data      []byte
		algorithm *compression.CompressionAlgorithm
	}{
		{
			name:      "Compress and decompress with Gzip",
			data:      []byte("This is a test string for compression and decompression"),
			algorithm: compression.Gzip,
		},
		{
			name:      "Compress and decompress empty data with Gzip",
			data:      []byte{},
			algorithm: compression.Gzip,
		},
		{
			name:      "Compress and decompress large data with Gzip",
			data:      []byte(strings.Repeat("Large data test ", 1000)),
			algorithm: compression.Gzip,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			// Compress the data
			compressed, err := c.Compress(&testCase.data, testCase.algorithm)
			require.NoError(t, err)
			require.NotNil(t, compressed)

			// Decompress the data
			filename := "test" + testCase.algorithm.Extension
			decompressed, err := c.Decompress(&compressed, filename)
			require.NoError(t, err)
			require.NotNil(t, decompressed)

			// Check if the decompressed data matches the original
			assert.Equal(t, testCase.data, decompressed, "Decompressed data should match original data")
		})
	}
}
