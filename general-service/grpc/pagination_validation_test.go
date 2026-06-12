package grpc

import (
	"testing"

	pb "github.com/qubic/qubic-aggregation/general-service/api/qubic/aggregation/general/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageSizeLimits_ValidatePagination(t *testing.T) {
	const (
		maxPageSize     = 1000
		defaultPageSize = 10
		maxHits         = 10000
	)
	limits := NewPageSizeLimits(maxPageSize, defaultPageSize, maxHits)

	tests := []struct {
		name           string
		pagination     *pb.Pagination
		expectErr      bool
		expectedOffset uint32
		expectedSize   uint32
	}{
		{
			name:           "nil pagination uses defaults",
			pagination:     nil,
			expectedOffset: 0,
			expectedSize:   defaultPageSize,
		},
		{
			name:           "zero size uses default but keeps offset",
			pagination:     &pb.Pagination{Offset: 5, Size: 0},
			expectedOffset: 5,
			expectedSize:   defaultPageSize,
		},
		{
			name:           "size within range is kept",
			pagination:     &pb.Pagination{Offset: 20, Size: 50},
			expectedOffset: 20,
			expectedSize:   50,
		},
		{
			name:           "size at max is allowed",
			pagination:     &pb.Pagination{Offset: 0, Size: maxPageSize},
			expectedOffset: 0,
			expectedSize:   maxPageSize,
		},
		{
			name:       "size above max errors",
			pagination: &pb.Pagination{Offset: 0, Size: maxPageSize + 1},
			expectErr:  true,
		},
		{
			name:       "offset above maxHits errors",
			pagination: &pb.Pagination{Offset: maxHits + 1, Size: 10},
			expectErr:  true,
		},
		{
			name:       "offset plus size above maxHits errors",
			pagination: &pb.Pagination{Offset: maxHits - 5, Size: 10},
			expectErr:  true,
		},
		{
			name:           "offset plus size at maxHits boundary is allowed",
			pagination:     &pb.Pagination{Offset: maxHits - 10, Size: 10},
			expectedOffset: maxHits - 10,
			expectedSize:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, size, err := limits.ValidatePagination(tt.pagination)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOffset, offset)
			assert.Equal(t, tt.expectedSize, size)
		})
	}
}
