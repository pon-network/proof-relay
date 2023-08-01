package signing

import (
	"errors"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	bls "github.com/pon-pbs/bbRelay/bls"
)

func (f *ForkVersion) FromSlice(s []byte) error {
	if len(s) != 4 {
		return errors.New("invalid fork version length")
	}
	copy(f[:], s)
	return nil
}

var (
	DomainBuilder Domain

	DomainTypeBeaconProposer = DomainType{0x00, 0x00, 0x00, 0x00}
	DomainTypeAppBuilder     = DomainType{0x00, 0x00, 0x00, 0x01}
)

type SigningData struct {
	Root   Root   `ssz-size:"32"`
	Domain Domain `ssz-size:"32"`
}

type ForkData struct {
	CurrentVersion        ForkVersion `ssz-size:"4"`
	GenesisValidatorsRoot Root        `ssz-size:"32"`
}

type HashTreeRoot interface {
	HashTreeRoot() ([32]byte, error)
}

func ComputeDomain(domainType DomainType, forkVersionHex, genesisValidatorsRootHex string) (domain Domain, err error) {
	genesisValidatorsRoot := Root(common.HexToHash(genesisValidatorsRootHex))
	forkVersionBytes, err := hexutil.Decode(forkVersionHex)
	if err != nil || len(forkVersionBytes) != 4 {
		return domain, errors.New("Wrong Fork Version")
	}
	var forkVersion [4]byte
	copy(forkVersion[:], forkVersionBytes[:4])
	return ComputeSSZDomain(domainType, forkVersion, genesisValidatorsRoot), nil
}

func ComputeSSZDomain(dt DomainType, forkVersion ForkVersion, genesisValidatorsRoot Root) [32]byte {
	forkDataRoot, _ := (&ForkData{
		CurrentVersion:        forkVersion,
		GenesisValidatorsRoot: genesisValidatorsRoot,
	}).HashTreeRoot()

	var domain [32]byte
	copy(domain[0:4], dt[:])
	copy(domain[4:], forkDataRoot[0:28])

	return domain
}

func ComputeSigningRoot(obj HashTreeRoot, d Domain) ([32]byte, error) {
	var zero [32]byte
	root, err := obj.HashTreeRoot()
	if err != nil {
		return zero, err
	}
	signingData := SigningData{root, d}
	msg, err := signingData.HashTreeRoot()
	if err != nil {
		return zero, err
	}
	return msg, nil
}

func SignMessage(obj HashTreeRoot, d Domain, sk *bls.SecretKey) (phase0.BLSSignature, error) {
	root, err := ComputeSigningRoot(obj, d)
	if err != nil {
		return phase0.BLSSignature{}, err
	}

	signatureBytes := bls.Sign(sk, root[:]).Compress()

	var signature phase0.BLSSignature

	copy(signature[:], signatureBytes)

	return signature, nil
}

func VerifySignature(obj HashTreeRoot, d Domain, pkBytes, sigBytes []byte) (bool, error) {
	msg, err := ComputeSigningRoot(obj, d)
	if err != nil {
		return false, err
	}

	return bls.VerifySignatureBytes(msg[:], sigBytes, pkBytes)
}

func (h Root) String() string {
	return hexutil.Bytes(h[:]).String()
}
