package compression

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// ErrUnsupportedAlgorithm is returned when an unsupported compression algorithm is specified.
// It is always wrapped with the offending value, so callers should match it with errors.Is.
var ErrUnsupportedAlgorithm = errors.New("unsupported compression algorithm")

// CompressionAlgorithm represents the type of compression algorithm.
type CompressionAlgorithm struct {
	Name            string
	Extension       string
	ContentEncoding string
}

var (
	Zstd = &CompressionAlgorithm{
		Name:            "zstd",
		Extension:       ".zst",
		ContentEncoding: "zstd",
	}
	Gzip = &CompressionAlgorithm{
		Name:            "gzip",
		Extension:       ".gz",
		ContentEncoding: "gzip",
	}
	None = &CompressionAlgorithm{
		Name:            "none",
		Extension:       "",
		ContentEncoding: "identity",
	}
)

// Default is the algorithm new objects are written with.
var Default = Zstd

// decodable lists the algorithms that leave a recognisable extension behind, newest first.
// None is deliberately absent: its empty extension matches every filename.
var decodable = []*CompressionAlgorithm{Zstd, Gzip}

// Compressor provides methods for compressing and decompressing data.
//
// A Compressor is safe for concurrent use and is meant to be built once and shared: the zstd
// encoder and decoder it holds are expensive to construct but cheap to reuse.
type Compressor struct {
	// zstdEncoder and zstdDecoder serve the byte-slice API only. Both are concurrency-safe for
	// EncodeAll/DecodeAll and hold no per-call state; streaming needs its own instance per
	// stream because the sliding window is per-stream.
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
	// zstdErr records a construction failure so the byte-slice API can report it on use rather
	// than forcing every caller to handle a constructor error.
	zstdErr error

	gzipWriters sync.Pool
}

// NewCompressor creates a new Compressor instance.
func NewCompressor() *Compressor {
	c := &Compressor{
		gzipWriters: sync.Pool{
			New: func() any {
				return gzip.NewWriter(io.Discard)
			},
		},
	}

	// A nil destination yields an encoder usable for EncodeAll only, which is all the
	// byte-slice API needs. Zero frames keep an empty payload encoding to a real frame instead
	// of nothing at all, so a stored object always parses as zstd.
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithZeroFrames(true),
	)
	if err != nil {
		c.zstdErr = fmt.Errorf("failed to create zstd encoder: %w", err)

		return c
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		c.zstdErr = fmt.Errorf("failed to create zstd decoder: %w", err)

		return c
	}

	c.zstdEncoder = encoder
	c.zstdDecoder = decoder

	return c
}

// NewWriter returns a writer that compresses everything written to it into dst using the given
// algorithm. The caller must Close the returned writer to flush the trailing frame; closing it
// never closes dst.
func (c *Compressor) NewWriter(algorithm *CompressionAlgorithm, dst io.Writer) (io.WriteCloser, error) {
	if algorithm == nil {
		return nil, fmt.Errorf("%w: <nil>", ErrUnsupportedAlgorithm)
	}

	if dst == nil {
		return nil, errors.New("destination writer is nil")
	}

	switch algorithm.Name {
	case Zstd.Name:
		// A stream cannot share the encoder held on the Compressor: the sliding window belongs
		// to the stream. Concurrency is pinned to 1 so a writer costs one window, not one per
		// core, which matters when many streams are open at once.
		w, err := zstd.NewWriter(dst,
			zstd.WithEncoderLevel(zstd.SpeedFastest),
			zstd.WithEncoderConcurrency(1),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd writer: %w", err)
		}

		return w, nil
	case Gzip.Name:
		w, ok := c.gzipWriters.Get().(*gzip.Writer)
		if !ok {
			w = gzip.NewWriter(dst)
		}

		w.Reset(dst)

		return &pooledGzipWriter{Writer: w, pool: &c.gzipWriters}, nil
	case None.Name:
		return nopWriteCloser{dst}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algorithm.Name)
	}
}

// NewReader returns a reader that decompresses src using the given algorithm. The caller must
// Close the returned reader; closing it never closes src.
func (c *Compressor) NewReader(algorithm *CompressionAlgorithm, src io.Reader) (io.ReadCloser, error) {
	if algorithm == nil {
		return nil, fmt.Errorf("%w: <nil>", ErrUnsupportedAlgorithm)
	}

	if src == nil {
		return nil, errors.New("source reader is nil")
	}

	switch algorithm.Name {
	case Zstd.Name:
		// Streaming decode keeps per-stream state, so it gets its own decoder. IOReadCloser
		// hands back a Close that releases the decoder's goroutines.
		r, err := zstd.NewReader(src, zstd.WithDecoderConcurrency(1))
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}

		return r.IOReadCloser(), nil
	case Gzip.Name:
		r, err := gzip.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}

		return r, nil
	case None.Name:
		return io.NopCloser(src), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algorithm.Name)
	}
}

// Compress compresses the input data using the specified algorithm.
func (c *Compressor) Compress(data *[]byte, algorithm *CompressionAlgorithm) ([]byte, error) {
	if data == nil {
		return nil, errors.New("data is nil")
	}

	if algorithm == nil {
		return nil, fmt.Errorf("%w: <nil>", ErrUnsupportedAlgorithm)
	}

	// The shared encoder handles whole slices without allocating a window per call, so zstd
	// does not go through NewWriter here.
	if algorithm.Name == Zstd.Name {
		if c.zstdErr != nil {
			return nil, c.zstdErr
		}

		return c.zstdEncoder.EncodeAll(*data, nil), nil
	}

	var buf bytes.Buffer

	w, err := c.NewWriter(algorithm, &buf)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(*data); err != nil {
		// Close anyway so a pooled writer is returned rather than dropped.
		_ = w.Close()

		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decompress decompresses the input data, choosing the algorithm from the filename's extension.
func (c *Compressor) Decompress(data *[]byte, filename string) ([]byte, error) {
	if data == nil {
		return nil, errors.New("data is nil")
	}

	algorithm, err := GetCompressionAlgorithm(filename)
	if err != nil {
		return nil, err
	}

	if algorithm.Name == Zstd.Name {
		if c.zstdErr != nil {
			return nil, c.zstdErr
		}

		// A non-nil destination keeps an empty payload decoding to an empty slice rather than
		// a nil one, matching what the reader-based path returns.
		return c.zstdDecoder.DecodeAll(*data, []byte{})
	}

	r, err := c.NewReader(algorithm, bytes.NewReader(*data))
	if err != nil {
		return nil, err
	}

	defer r.Close()

	return io.ReadAll(r)
}

// AddExtension adds the compression extension to the filename if it's not already present.
func AddExtension(filename string, algorithm *CompressionAlgorithm) string {
	if algorithm == nil {
		return filename
	}

	if !strings.HasSuffix(filename, algorithm.Extension) {
		return filename + algorithm.Extension
	}

	return filename
}

// RemoveExtension removes the compression extension from the filename if it's present.
func RemoveExtension(filename string) string {
	for _, algorithm := range decodable {
		if trimmed := strings.TrimSuffix(filename, algorithm.Extension); trimmed != filename {
			return trimmed
		}
	}

	return filename
}

// HasCompressionExtension checks if the filename has the compression extension. A nil algorithm,
// or one with no extension of its own, matches nothing.
func HasCompressionExtension(filename string, algorithm *CompressionAlgorithm) bool {
	if algorithm == nil || algorithm.Extension == "" {
		return false
	}

	return strings.HasSuffix(filename, algorithm.Extension)
}

// HasAnyCompressionExtension checks if the filename carries any compression extension.
func HasAnyCompressionExtension(filename string) bool {
	for _, algorithm := range decodable {
		if strings.HasSuffix(filename, algorithm.Extension) {
			return true
		}
	}

	return false
}

// GetCompressionAlgorithm resolves the algorithm a filename's extension implies.
func GetCompressionAlgorithm(filename string) (*CompressionAlgorithm, error) {
	for _, algorithm := range decodable {
		if strings.HasSuffix(filename, algorithm.Extension) {
			return algorithm, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, filename)
}

// GetCompressionAlgorithmFromContentEncoding resolves the algorithm a Content-Encoding names.
func GetCompressionAlgorithmFromContentEncoding(contentEncoding string) (*CompressionAlgorithm, error) {
	for _, algorithm := range []*CompressionAlgorithm{Zstd, Gzip, None} {
		if contentEncoding == algorithm.ContentEncoding {
			return algorithm, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, contentEncoding)
}

// pooledGzipWriter returns its underlying writer to the pool on Close, so a compressor that is
// called repeatedly does not allocate a fresh window each time.
type pooledGzipWriter struct {
	*gzip.Writer

	pool *sync.Pool
}

func (w *pooledGzipWriter) Close() error {
	if w.Writer == nil {
		return nil
	}

	gz := w.Writer
	w.Writer = nil

	err := gz.Close()

	// Drop the reference to the destination before parking the writer, so a pooled writer
	// cannot keep a finished stream alive.
	gz.Reset(io.Discard)
	w.pool.Put(gz)

	return err
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
