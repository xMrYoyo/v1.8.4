package main

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type p2pConverter struct{}

// NewP2pConverter creates a new instance of p2p converter
func NewP2pConverter() *p2pConverter {
	return &p2pConverter{}
}

// Len return zero
func (p *p2pConverter) Len() int {
	return 0
}

// Decode does nothing
func (p *p2pConverter) Decode(humanReadable string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Encode encodes a byte array representing public key as peer ID string
func (p *p2pConverter) Encode(pkBytes []byte) (string, error) {
	pubKey, err := libp2pCrypto.UnmarshalSecp256k1PublicKey(pkBytes)
	if err != nil {
		return "", err
	}

	id, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return "", err
	}

	return id.Pretty(), nil
}

// EncodSlice encodes a byte array slice representing public key as peer ID string
func (p *p2pConverter) EncodeSlice(pkBytesSlice [][]byte) ([]string, error) {
	encodedSlice := make([]string, 0, len(pkBytesSlice))

	for _, pkBytes := range pkBytesSlice {
		pubKey, err := libp2pCrypto.UnmarshalSecp256k1PublicKey(pkBytes)
		if err != nil {
			return nil, err
		}

		id, err := peer.IDFromPublicKey(pubKey)
		if err != nil {
			return nil, err
		}

		encodedSlice = append(encodedSlice, id.Pretty())
	}
	return encodedSlice, nil
}

// Encode encodes a byte array representing public key as peer ID string
func (p *p2pConverter) SilentEncode(pkBytes []byte, log core.Logger) string {
	pubKey, err := libp2pCrypto.UnmarshalSecp256k1PublicKey(pkBytes)
	if err != nil {
		log.Warn("err")
		return ""
	}

	id, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		log.Warn("err")
		return ""
	}

	return id.Pretty()
}

// IsInterfaceNil returns true if there is no value under the interface
func (p *p2pConverter) IsInterfaceNil() bool {
	return p == nil
}
