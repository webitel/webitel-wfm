package pkg

func ToPTR[T any](s T) *T {
	return &s
}
