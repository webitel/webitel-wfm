package encoding

import (
	"bytes"
	"encoding/gob"
)

type gobSerializer struct{}

func (s *gobSerializer) Serialize(v any) ([]byte, error) {
	var buffer bytes.Buffer

	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(v); err != nil {
		return []byte{}, err
	}

	return buffer.Bytes(), nil
}

func (s *gobSerializer) Deserialize(data []byte, v any) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	return dec.Decode(v)
}
