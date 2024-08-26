package model

type Field struct {
	Default bool
	Column  string
}

type FieldsFormatter interface {
	Fields() []Field
}
