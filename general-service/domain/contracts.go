package domain

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"github.com/qubic/go-node-connector/types"
)

func ContractIndexToAddress(contractIndex uint32) (string, error) {
	var pubKey [32]byte
	binary.LittleEndian.PutUint32(pubKey[:4], contractIndex)

	var identity types.Identity
	id, err := identity.FromPubKey(pubKey, false)
	if err != nil {
		return "", fmt.Errorf("deriving identity from contract index %d: %w", contractIndex, err)
	}

	return string(id), nil
}

func ParseBidInputData(inputData string) (IpoBid, error) {
	data, err := base64.StdEncoding.DecodeString(inputData)
	if err != nil {
		return IpoBid{}, fmt.Errorf("decoding base64 input data: %w", err)
	}

	if len(data) != 16 {
		return IpoBid{}, fmt.Errorf("unexpected input data size: got %d, expected 16", len(data))
	}

	price := int64(binary.LittleEndian.Uint64(data[0:8]))
	quantity := binary.LittleEndian.Uint16(data[8:10])
	// bytes 10-15 are struct padding, discarded

	return IpoBid{
		Price:    price,
		Quantity: quantity,
	}, nil
}
