package pagination

import "fmt"

type Limits struct {
	MaxPageSize     uint32
	DefaultPageSize uint32
	MaxOffset       uint32
}

func DefaultLimits() Limits {
	return Limits{MaxPageSize: 1000, DefaultPageSize: 10, MaxOffset: 10000}
}

func (l Limits) Normalize(offset, size uint32) (uint32, uint32, error) {
	if size == 0 {
		size = l.DefaultPageSize
	}
	if size > l.MaxPageSize {
		return 0, 0, fmt.Errorf("page size %d exceeds maximum %d", size, l.MaxPageSize)
	}
	if offset > l.MaxOffset {
		return 0, 0, fmt.Errorf("offset %d exceeds maximum %d", offset, l.MaxOffset)
	}
	if offset+size > l.MaxOffset {
		return 0, 0, fmt.Errorf("offset + size (%d) exceeds maximum %d", offset+size, l.MaxOffset)
	}
	return offset, size, nil
}
