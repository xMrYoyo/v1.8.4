package hashing

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-sandbox/hashing/keccak"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing/sha256"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/pkg/errors"
)

// ErrNotImplemented signals that a functionality can not be used as it is not implemented
var ErrNotImplemented = errors.New("not implemented")

// VMCrypto is a wrapper used in vm implementation
type VMCrypto struct {
}

// Sha256 returns a sha 256 hash of the input string
func (vmc *VMCrypto) Sha256(str string) (string, error) {
	return string(sha256.Sha256{}.Compute(str)), nil
}

// Keccak256 returns a keccak 256 hash of the input string
func (vmc *VMCrypto) Keccak256(str string) (string, error) {
	return string(keccak.Keccak{}.Compute(str)), nil
}

// Ripemd160 is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) Ripemd160(str string) (string, error) {
	return "", ErrNotImplemented
}

// EcdsaRecover is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) EcdsaRecover(hash string, v *big.Int, r string, s string) (string, error) {
	return "", ErrNotImplemented
}

// Bn128valid is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) Bn128valid(p vmcommon.Bn128Point) (bool, error) {
	return false, ErrNotImplemented
}

// Bn128g2valid is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) Bn128g2valid(p vmcommon.Bn128G2Point) (bool, error) {
	return false, ErrNotImplemented
}

// Bn128add is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) Bn128add(p1 vmcommon.Bn128Point, p2 vmcommon.Bn128Point) (vmcommon.Bn128Point, error) {
	return vmcommon.Bn128Point{}, ErrNotImplemented
}

// Bn128mul is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) Bn128mul(k *big.Int, p vmcommon.Bn128Point) (vmcommon.Bn128Point, error) {
	return vmcommon.Bn128Point{}, ErrNotImplemented
}

// Bn128ate is deprecated and should be removed as soon as the vmcommon interface has been updated
//TODO remove this when the vmcommon interface has been updated
func (vmc *VMCrypto) Bn128ate(l1 []vmcommon.Bn128Point, l2 []vmcommon.Bn128G2Point) (bool, error) {
	return false, ErrNotImplemented
}
