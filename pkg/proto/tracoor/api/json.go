package api

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON for Entity enum
func (e Entity) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

// UnmarshalJSON for Entity enum
func (e *Entity) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Entity should be a string, got %s", data)
	}

	if val, ok := Entity_value[s]; ok {
		*e = Entity(val)
		return nil
	}
	return fmt.Errorf("unexpected entity value: %s", s)
}

// MarshalJSON for Field enum
func (f ListUniqueValuesRequest_Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

// UnmarshalJSON for Field enum
func (f *ListUniqueValuesRequest_Field) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Field should be a string, got %s", data)
	}

	if val, ok := ListUniqueValuesRequest_Field_value[s]; ok {
		*f = ListUniqueValuesRequest_Field(val)
		return nil
	}
	return fmt.Errorf("unexpected field value: %s", s)
}
