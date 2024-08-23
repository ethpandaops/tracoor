package mime

import (
	"strings"
)

// ContentType represents the MIME type of a file.
type ContentType string

const (
	ContentTypeUnknown ContentType = "application/unknown"
	ContentTypeJSON    ContentType = "application/json"
	ContentTypeOctet   ContentType = "application/octet-stream"
)

func ContentTypeFromString(s string) ContentType {
	if s == "" {
		return ContentTypeUnknown
	}

	return ContentType(s)
}

// GetContentTypeFromExtension returns the MIME type of a file based on its extension.
func GetContentTypeFromExtension(extension string) ContentType {
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
