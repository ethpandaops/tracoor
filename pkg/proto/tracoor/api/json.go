package api

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON for Field enum
func (f ListUniqueBeaconStateValuesRequest_Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

// UnmarshalJSON for Field enum
func (f *ListUniqueBeaconStateValuesRequest_Field) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Field should be a string, got %s", data)
	}

	if val, ok := ListUniqueBeaconStateValuesRequest_Field_value[s]; ok {
		*f = ListUniqueBeaconStateValuesRequest_Field(val)
		return nil
	}
	return fmt.Errorf("unexpected field value: %s", s)
}