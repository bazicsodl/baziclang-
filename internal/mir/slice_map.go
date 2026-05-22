package mir

func mapSlice[T any, U any](items []T, f func(T) U) []U {
	if len(items) == 0 {
		return nil
	}
	out := make([]U, 0, len(items))
	for _, item := range items {
		out = append(out, f(item))
	}
	return out
}

func mapSliceE[T any, U any](items []T, f func(T) (U, error)) ([]U, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]U, 0, len(items))
	for _, item := range items {
		mapped, err := f(item)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func mapOptional[T comparable, U any](item T, zero T, f func(T) U) U {
	if item == zero {
		var out U
		return out
	}
	return f(item)
}

func mapOptionalE[T comparable, U any](item T, zero T, f func(T) (U, error)) (U, error) {
	if item == zero {
		var out U
		return out, nil
	}
	return f(item)
}
