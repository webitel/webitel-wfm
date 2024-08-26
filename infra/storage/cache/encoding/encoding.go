package encoding

// Serializer for data
type Serializer interface {
	// Serialize the data to byte array
	Serialize(v any) ([]byte, error)
	// Deserialize the byte array to destination value
	Deserialize(data []byte, v any) error
}

// DefaultSerializer the default Serializer implementation
var DefaultSerializer Serializer = &gobSerializer{}
