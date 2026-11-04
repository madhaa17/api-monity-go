package port

import "encoding/json"

// Optional is a JSON field of a partial update that tells "absent" (leave unchanged) apart from
// null (clear the value): Set is true whenever the key was present, and Value is nil for null.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
