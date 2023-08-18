package spec

type TypedList[T any] struct {
	list    List
	element func([]byte) (T, int, error)
}

// NewTypedList returns a typed list or an empty list on error.
func NewTypedList[T any](b []byte, element func([]byte) (T, int, error)) TypedList[T] {
	l, _, _ := decodeList(b)

	return TypedList[T]{
		list:    l,
		element: element,
	}
}

// ParseTypedList decodes, recursively validates and returns a list.
func ParseTypedList[T any](b []byte, element func([]byte) (T, int, error)) (_ TypedList[T], size int, err error) {
	l, size, err := decodeList(b)
	if err != nil {
		return
	}

	list := TypedList[T]{
		list:    l,
		element: element,
	}

	ln := l.Len()
	for i := 0; i < ln; i++ {
		b1 := l.GetBytes(i)
		if len(b1) == 0 {
			continue
		}

		if _, _, err = element(b1); err != nil {
			return
		}
	}
	return list, size, nil
}

// Len returns the number of elements in the list.
func (l TypedList[T]) Len() int {
	return l.list.Len()
}

// Raw returns the exact list bytes.
func (l TypedList[T]) Raw() []byte {
	return l.list.bytes
}

// Empty returns true if bytes are empty or list has no elements.
func (l TypedList[T]) Empty() bool {
	return l.list.Empty()
}

// Get returns an element at index i, panics on out of range.
func (l TypedList[T]) Get(i int) T {
	b := l.list.GetBytes(i)
	elem, _, _ := l.element(b)
	return elem
}

// GetBytes returns element bytes at index i, panics on out of range.
func (l TypedList[T]) GetBytes(i int) []byte {
	return l.list.GetBytes(i)
}

// Values converts a list into a slice.
func (l TypedList[T]) Values() []T {
	result := make([]T, 0, l.list.Len())

	for i := 0; i < l.list.Len(); i++ {
		elem := l.Get(i)
		result = append(result, elem)
	}

	return result
}
