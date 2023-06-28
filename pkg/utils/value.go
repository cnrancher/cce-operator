package utils

type valueTypes interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 |
		~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64 | ~string | ~bool |
		[]string
}

// GetPtr gets the pointer of the variable
func GetPtr[T valueTypes](i T) *T {
	return &i
}

// GetPtr is a safe function to get the data from the pointer
func GetValue[T valueTypes](p *T) T {
	if p == nil {
		return *new(T)
	}
	return *p
}
