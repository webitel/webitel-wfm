package options

type Derived struct {
	fields
	derived
	orderBy

	id int64

	//nolint:unused
	filter map[string]any
}

func (d *Derived) ID() int64 {
	return d.id
}

func (d *Derived) WithID(id int64) {
	d.id = id
}
