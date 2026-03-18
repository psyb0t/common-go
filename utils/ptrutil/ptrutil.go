package ptrutil

func Of[T any](v T) *T {
	return &v
}

func ZeroOrVal[T any](ptr *T) T { //nolint:ireturn
	var zero T

	if ptr != nil {
		return *ptr
	}

	return zero
}

func DefaultOrVal[T any](ptr *T, defaultVal T) T { //nolint:ireturn
	if ptr != nil {
		return *ptr
	}

	return defaultVal
}
