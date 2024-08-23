package mime_test

import (
	"testing"

	"github.com/ethpandaops/tracoor/pkg/mime"
)

func TestGetContentTypeFromExtension(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		want      mime.ContentType
	}{
		{
			name:      "JSON extension",
			extension: "json",
			want:      mime.ContentTypeJSON,
		},
		{
			name:      "SSZ extension",
			extension: "ssz",
			want:      mime.ContentTypeOctet,
		},
		{
			name:      "Unknown extension",
			extension: "txt",
			want:      mime.ContentTypeUnknown,
		},
		{
			name:      "Empty extension",
			extension: "",
			want:      mime.ContentTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mime.GetContentTypeFromExtension(tt.extension)
			if got != tt.want {
				t.Errorf("GetContentTypeFromExtension(%q) = %v, want %v", tt.extension, got, tt.want)
			}
		})
	}
}
