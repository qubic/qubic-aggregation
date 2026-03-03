package domain

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractIndexToAddress(t *testing.T) {
	tests := []struct {
		name  string
		index uint32
	}{
		{name: "index 0", index: 0},
		{name: "index 1", index: 1},
		{name: "index 5", index: 5},
		{name: "large index", index: 1000},
	}

	seen := make(map[string]uint32)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := ContractIndexToAddress(tt.index)
			require.NoError(t, err)
			assert.Len(t, addr, 60, "identity should be 60 characters")

			if prev, ok := seen[addr]; ok {
				t.Errorf("index %d produced same address as index %d", tt.index, prev)
			}
			seen[addr] = tt.index
		})
	}
}

func TestContractIndexToAddress_KnownAddresses(t *testing.T) {
	known := []struct {
		index   uint32
		name    string
		address string
	}{
		{1, "QX", "BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAARMID"},
		{2, "QTRY", "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACNKL"},
		{3, "RANDOM", "DAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAANMIG"},
		{4, "QUTIL", "EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF"},
		{5, "MLM", "FAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAYWJB"},
		{6, "GQMPROP", "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQGNM"},
		{7, "SWATCH", "HAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHYCM"},
	}

	for _, tt := range known {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := ContractIndexToAddress(tt.index)
			require.NoError(t, err)
			assert.Equal(t, tt.address, addr)
		})
	}
}

func TestParseBidInputData(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		var buf [10]byte
		binary.LittleEndian.PutUint64(buf[0:8], uint64(1000))
		binary.LittleEndian.PutUint16(buf[8:10], 5)
		encoded := base64.StdEncoding.EncodeToString(buf[:])

		bid, err := ParseBidInputData(encoded)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), bid.Price)
		assert.Equal(t, uint16(5), bid.Quantity)
	})

	t.Run("max values", func(t *testing.T) {
		var buf [10]byte
		binary.LittleEndian.PutUint64(buf[0:8], ^uint64(0)>>1) // max int64
		binary.LittleEndian.PutUint16(buf[8:10], 0xFFFF)
		encoded := base64.StdEncoding.EncodeToString(buf[:])

		bid, err := ParseBidInputData(encoded)
		require.NoError(t, err)
		assert.Equal(t, int64(1<<63-1), bid.Price)
		assert.Equal(t, uint16(0xFFFF), bid.Quantity)
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := ParseBidInputData("not-valid-base64!!!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decoding base64")
	})

	t.Run("wrong length short", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03})
		_, err := ParseBidInputData(encoded)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected input data size")
	})

	t.Run("wrong length long", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString(make([]byte, 12))
		_, err := ParseBidInputData(encoded)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected input data size")
	})
}
