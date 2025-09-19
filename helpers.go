package confgo

func ptr[T any](val T) *T {
	return &val
}
