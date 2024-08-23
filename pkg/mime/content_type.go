package mime

import (
	"fmt"
	"strings"
)

// ContentType represents the MIME type of a file.
type ContentType string

const (
	ContentTypeUnknown ContentType = "application/unknown"
	ContentTypeJSON    ContentType = "application/json"
	ContentTypeOctet   ContentType = "application/octet-stream"
)

// GetContentTypeFromExtension returns the MIME type of a file based on its extension.
func GetContentTypeFromExtension(extension string) ContentType {
	fmt.Println("Extension: ", extension)
	if extension == "" {
		return ContentTypeUnknown
	}

	extension = strings.TrimPrefix(extension, ".")

	if extension == "json" {
		return ContentTypeJSON
	}

	if extension == "ssz" {
		return ContentTypeOctet
	}

	return ContentTypeUnknown
}
