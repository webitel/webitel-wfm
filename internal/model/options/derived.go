package options

type Derived struct {
	fields
	derived
	orderBy

	//nolint:unused
	filter map[string]any
}
