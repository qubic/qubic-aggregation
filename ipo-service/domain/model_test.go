package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEpochIntervalsAbsoluteRange(t *testing.T) {
	t.Run("empty intervals", func(t *testing.T) {
		first, last := GetEpochIntervalsAbsoluteRange(nil)
		assert.Equal(t, uint32(0), first)
		assert.Equal(t, uint32(0), last)
	})

	t.Run("single interval", func(t *testing.T) {
		intervals := []TickInterval{{First: 100, Last: 200}}
		first, last := GetEpochIntervalsAbsoluteRange(intervals)
		assert.Equal(t, uint32(100), first)
		assert.Equal(t, uint32(200), last)
	})

	t.Run("multiple contiguous intervals", func(t *testing.T) {
		intervals := []TickInterval{
			{First: 100, Last: 200},
			{First: 201, Last: 300},
			{First: 301, Last: 400},
		}
		first, last := GetEpochIntervalsAbsoluteRange(intervals)
		assert.Equal(t, uint32(100), first)
		assert.Equal(t, uint32(400), last)
	})

	t.Run("non-overlapping unordered intervals", func(t *testing.T) {
		intervals := []TickInterval{
			{First: 500, Last: 600},
			{First: 100, Last: 200},
			{First: 300, Last: 400},
		}
		first, last := GetEpochIntervalsAbsoluteRange(intervals)
		assert.Equal(t, uint32(100), first)
		assert.Equal(t, uint32(600), last)
	})
}
