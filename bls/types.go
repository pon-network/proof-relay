package bls

import (
	"errors"

	blst "github.com/supranational/blst/bindings/go"
)

var dst = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

const (
	BLSPublicKeyLength int = 48
	BLSSecretKeyLength int = 32
	BLSSignatureLength int = 96
)

type (
	PublicKey = blst.P1Affine
	SecretKey = blst.SecretKey
	Signature = blst.P2Affine
)

var (
	ErrDeserializeSecretKey   = errors.New("could not deserialize secret key from bytes")
	ErrInvalidPubkey          = errors.New("invalid pubkey")
	ErrInvalidPubkeyLength    = errors.New("invalid pubkey length")
	ErrInvalidSecretKeyLength = errors.New("invalid secret key length")
	ErrInvalidSignature       = errors.New("invalid signature")
	ErrInvalidSignatureLength = errors.New("invalid signature length")
	ErrUncompressPubkey       = errors.New("could not uncompress public key from bytes")
	ErrUncompressSignature    = errors.New("could not uncompress signature from bytes")
)
