package encoding

import "encoding/json"

//nolint:unused
type jsonSerializer struct{}

//nolint:unused
func (s *jsonSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

//nolint:unused
func (s *jsonSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, &v)
}
