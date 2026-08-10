package compression_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/tracoor/pkg/compression"
)

const (
	testFilename    = "test"
	testFilenameGz  = "test.gz"
	testFilenameZst = "test.zst"
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
			name:      "Compress with Zstd",
			data:      []byte("test data"),
			algorithm: compression.Zstd,
			wantErr:   false,
		},
		{
			name:      "Compress with nil algorithm",
			data:      []byte("test data"),
			algorithm: nil,
			wantErr:   true,
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

	compressedZstd, err := c.Compress(&testData, compression.Zstd)
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
			filename: testFilenameGz,
			wantErr:  false,
		},
		{
			name:     "Decompress Zstd",
			data:     compressedZstd,
			filename: testFilenameZst,
			wantErr:  false,
		},
		{
			name:     "Decompress with nil data",
			data:     nil,
			filename: testFilenameGz,
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
			filename:  testFilename,
			algorithm: compression.Gzip,
			want:      "test.gz",
		},
		{
			name:      "Extension already present",
			filename:  testFilenameGz,
			algorithm: compression.Gzip,
			want:      "test.gz",
		},
		{
			name:      "Add Zstd extension",
			filename:  testFilename,
			algorithm: compression.Zstd,
			want:      "test.zst",
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
			filename:  testFilenameGz,
			algorithm: compression.Gzip,
			want:      "test",
		},
		{
			name:      "No extension to remove",
			filename:  testFilename,
			algorithm: compression.Gzip,
			want:      "test",
		},
		{
			name:      "Remove Gzip extension from .json.gz",
			filename:  "test.json.gz",
			algorithm: compression.Gzip,
			want:      "test.json",
		},
		{
			name:      "Remove Zstd extension",
			filename:  testFilenameZst,
			algorithm: compression.Zstd,
			want:      "test",
		},
		{
			name:      "Remove Zstd extension from .ssz.zst",
			filename:  "test.ssz.zst",
			algorithm: compression.Zstd,
			want:      "test.ssz",
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
			filename:  testFilenameGz,
			algorithm: compression.Gzip,
			want:      true,
		},
		{
			name:      "No Gzip extension",
			filename:  testFilename,
			algorithm: compression.Gzip,
			want:      false,
		},
		{
			name:      "Has Zstd extension",
			filename:  testFilenameZst,
			algorithm: compression.Zstd,
			want:      true,
		},
		{
			name:      "Gzip file is not Zstd",
			filename:  testFilenameGz,
			algorithm: compression.Zstd,
			want:      false,
		},
		{
			// A nil algorithm reaches this function whenever a lookup failed upstream. It must
			// answer rather than panic.
			name:      "Nil algorithm",
			filename:  testFilenameGz,
			algorithm: nil,
			want:      false,
		},
		{
			// None has no extension of its own, so it must not match every filename.
			name:      "None algorithm never matches",
			filename:  testFilenameGz,
			algorithm: compression.None,
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
			filename: testFilenameGz,
			want:     compression.Gzip,
			wantErr:  false,
		},
		{
			name:     "Get Zstd algorithm",
			filename: testFilenameZst,
			want:     compression.Zstd,
			wantErr:  false,
		},
		{
			name:     "Unsupported algorithm",
			filename: "test.unsupported",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "No extension at all",
			filename: testFilename,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			result, err := compression.GetCompressionAlgorithm(testCase.filename)
			if testCase.wantErr {
				require.ErrorIs(t, err, compression.ErrUnsupportedAlgorithm)
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
		{
			name:      "Compress and decompress with Zstd",
			data:      []byte("This is a test string for compression and decompression"),
			algorithm: compression.Zstd,
		},
		{
			name:      "Compress and decompress empty data with Zstd",
			data:      []byte{},
			algorithm: compression.Zstd,
		},
		{
			name:      "Compress and decompress large data with Zstd",
			data:      []byte(strings.Repeat("Large data test ", 1000)),
			algorithm: compression.Zstd,
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

func TestGetCompressionAlgorithmFromContentEncoding(t *testing.T) {
	testCases := []struct {
		name            string
		contentEncoding string
		want            *compression.CompressionAlgorithm
		wantErr         bool
	}{
		{
			name:            "Gzip encoding",
			contentEncoding: "gzip",
			want:            compression.Gzip,
		},
		{
			name:            "Zstd encoding",
			contentEncoding: "zstd",
			want:            compression.Zstd,
		},
		{
			name:            "Identity encoding",
			contentEncoding: "identity",
			want:            compression.None,
		},
		{
			name:            "Empty encoding",
			contentEncoding: "",
			wantErr:         true,
		},
		{
			name:            "Unknown encoding",
			contentEncoding: "brotli",
			wantErr:         true,
		},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			result, err := compression.GetCompressionAlgorithmFromContentEncoding(testCase.contentEncoding)
			if testCase.wantErr {
				require.ErrorIs(t, err, compression.ErrUnsupportedAlgorithm)
				assert.Nil(t, result)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestHasAnyCompressionExtension(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "Gzip", filename: testFilenameGz, want: true},
		{name: "Zstd", filename: testFilenameZst, want: true},
		{name: "Suffixed Zstd", filename: "test.ssz.zst", want: true},
		{name: "No extension", filename: testFilename, want: false},
		{name: "Uncompressed extension", filename: "test.ssz", want: false},
		{name: "Empty filename", filename: "", want: false},
	}

	for _, tc := range testCases {
		testCase := tc

		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, compression.HasAnyCompressionExtension(testCase.filename))
		})
	}
}

func TestCompressor_Streams(t *testing.T) {
	c := compression.NewCompressor()

	payloads := map[string][]byte{
		"empty":  {},
		"small":  []byte("This is a test string for compression and decompression"),
		"large":  []byte(strings.Repeat("Large data test ", 100000)),
		"binary": bytes.Repeat([]byte{0x00, 0xff, 0x7f, 0x01}, 4096),
	}

	algorithms := []*compression.CompressionAlgorithm{
		compression.Zstd,
		compression.Gzip,
		compression.None,
	}

	for _, algorithm := range algorithms {
		for name, payload := range payloads {
			algo, data := algorithm, payload

			t.Run(algo.Name+"/"+name, func(t *testing.T) {
				var buf bytes.Buffer

				w, err := c.NewWriter(algo, &buf)
				require.NoError(t, err)

				n, err := w.Write(data)
				require.NoError(t, err)
				require.Equal(t, len(data), n)
				require.NoError(t, w.Close())

				r, err := c.NewReader(algo, bytes.NewReader(buf.Bytes()))
				require.NoError(t, err)

				defer r.Close()

				got, err := io.ReadAll(r)
				require.NoError(t, err)
				assert.True(t, bytes.Equal(data, got), "round-tripped stream should match the original")
			})
		}
	}
}

func TestCompressor_StreamsInteroperateWithByteAPI(t *testing.T) {
	c := compression.NewCompressor()

	data := []byte(strings.Repeat("interop ", 5000))

	for _, algorithm := range []*compression.CompressionAlgorithm{compression.Zstd, compression.Gzip} {
		algo := algorithm

		t.Run(algo.Name, func(t *testing.T) {
			// Stream in, byte-slice out.
			var buf bytes.Buffer

			w, err := c.NewWriter(algo, &buf)
			require.NoError(t, err)

			_, err = w.Write(data)
			require.NoError(t, err)
			require.NoError(t, w.Close())

			streamed := buf.Bytes()

			decompressed, err := c.Decompress(&streamed, testFilename+algo.Extension)
			require.NoError(t, err)
			assert.Equal(t, data, decompressed)

			// Byte-slice in, stream out.
			compressed, err := c.Compress(&data, algo)
			require.NoError(t, err)

			r, err := c.NewReader(algo, bytes.NewReader(compressed))
			require.NoError(t, err)

			defer r.Close()

			got, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, data, got)
		})
	}
}

func TestCompressor_NewWriterAndReaderErrors(t *testing.T) {
	c := compression.NewCompressor()

	unsupported := &compression.CompressionAlgorithm{Name: "brotli", Extension: ".br", ContentEncoding: "br"}

	_, err := c.NewWriter(unsupported, &bytes.Buffer{})
	require.ErrorIs(t, err, compression.ErrUnsupportedAlgorithm)

	_, err = c.NewWriter(nil, &bytes.Buffer{})
	require.ErrorIs(t, err, compression.ErrUnsupportedAlgorithm)

	_, err = c.NewWriter(compression.Zstd, nil)
	require.Error(t, err)

	_, err = c.NewReader(unsupported, bytes.NewReader(nil))
	require.ErrorIs(t, err, compression.ErrUnsupportedAlgorithm)

	_, err = c.NewReader(nil, bytes.NewReader(nil))
	require.ErrorIs(t, err, compression.ErrUnsupportedAlgorithm)

	_, err = c.NewReader(compression.Zstd, nil)
	require.Error(t, err)
}

// TestCompressor_Concurrent exercises the shared zstd encoder and decoder, and the pooled gzip
// writers, from many goroutines at once. Run with -race.
func TestCompressor_Concurrent(t *testing.T) {
	c := compression.NewCompressor()

	const goroutines = 32

	algorithms := []*compression.CompressionAlgorithm{compression.Zstd, compression.Gzip}

	var wg sync.WaitGroup

	for i := range goroutines {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			algo := algorithms[id%len(algorithms)]
			data := []byte(strings.Repeat(fmt.Sprintf("goroutine-%d ", id), 2000))

			for range 10 {
				// Byte-slice API, which shares one encoder and one decoder.
				compressed, err := c.Compress(&data, algo)
				if err != nil {
					t.Errorf("compress: %v", err)

					return
				}

				decompressed, err := c.Decompress(&compressed, testFilename+algo.Extension)
				if err != nil {
					t.Errorf("decompress: %v", err)

					return
				}

				if !bytes.Equal(data, decompressed) {
					t.Errorf("byte-slice round trip mismatch for %s", algo.Name)

					return
				}

				// Stream API, which builds per-stream state.
				var buf bytes.Buffer

				w, err := c.NewWriter(algo, &buf)
				if err != nil {
					t.Errorf("new writer: %v", err)

					return
				}

				if _, err = w.Write(data); err != nil {
					t.Errorf("stream write: %v", err)

					return
				}

				if err = w.Close(); err != nil {
					t.Errorf("stream close: %v", err)

					return
				}

				r, err := c.NewReader(algo, bytes.NewReader(buf.Bytes()))
				if err != nil {
					t.Errorf("new reader: %v", err)

					return
				}

				got, err := io.ReadAll(r)

				if cerr := r.Close(); cerr != nil {
					t.Errorf("stream reader close: %v", cerr)

					return
				}

				if err != nil {
					t.Errorf("stream read: %v", err)

					return
				}

				if !bytes.Equal(data, got) {
					t.Errorf("stream round trip mismatch for %s", algo.Name)

					return
				}
			}
		}(i)
	}

	wg.Wait()
}
