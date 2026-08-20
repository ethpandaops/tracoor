package yaml

type RawMessage struct {
	unmarshal func(interface{}) error
}

func (r *RawMessage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	r.unmarshal = unmarshal

	return nil
}

func (r *RawMessage) Unmarshal(v interface{}) error {
	return r.unmarshal(v)
}

// IsZero reports whether the message was never populated, i.e. its key was
// absent from the document. Unmarshal panics on a zero message.
func (r *RawMessage) IsZero() bool {
	return r.unmarshal == nil
}
