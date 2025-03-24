package options

type Derived struct {
	fields
	derived
	orderBy

	filter map[string]any
}
