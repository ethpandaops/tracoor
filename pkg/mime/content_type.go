package mime

// ContentType represents the MIME type of a file.
type ContentType string

const (
	ContentTypeUnknown ContentType = "application/unknown"
	ContentTypeJSON    ContentType = "application/json"
	ContentTypeOctet   ContentType = "application/octet-stream"
)

// GetContentTypeFromExtension returns the MIME type of a file based on its extension.
func GetContentTypeFromExtension(extension string) ContentType {
	if extension == "json" {
		return ContentTypeJSON
	}

	if extension == "ssz" {
		return ContentTypeOctet
	}

	return ContentTypeUnknown
}
