package utils

import (
	"errors"
	"sync"
	"time"

	capellaAPI "github.com/attestantio/go-builder-client/api/capella"
	"github.com/attestantio/go-eth2-client/spec/capella"
	relayTypes "github.com/bsn-eng/pon-golang-types/relay"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
	"github.com/pon-pbs/bbRelay/redisPackage"
)

var EpochDuration = 12 * 32 * time.Second

var (
	keyValidatorStatus = "validator-status"
	keyBuilderStatus   = "builder-status"
	keyReporterrStatus = "reporter-status"
)

type PublicKey [48]byte

func (p *PublicKey) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(p[:])
	if err := b.UnmarshalText(input); err != nil {
		return err
	}
	return p.FromSlice(b)
}

func (p *PublicKey) FromSlice(x []byte) error {
	if len(x) != 48 {
		return errors.New("Wrong Length")
	}
	copy(p[:], x)
	return nil
}

func (p PublicKey) String() string {
	return hexutil.Bytes(p[:]).String()
}

type ProposerUtils struct {
	Log            logrus.Entry
	RedisInterface *redisPackage.RedisInterface
	BeaconClient   *beaconclient.MultiBeaconClient
	Validators     relayTypes.ValidatorIndexes
	ProposerStatus ProposerUpdates
}

type ProposerUpdates struct {
	ValidatorsLast map[string]string
	Mu             sync.Mutex
}

type BuilderUtils struct {
	BuilderLast    map[string]string
	Mu             sync.Mutex
	Log            logrus.Entry
	RedisInterface *redisPackage.RedisInterface
}

type ReporterUtils struct {
	ReporterLast   map[string]bool
	Mu             sync.Mutex
	Log            logrus.Entry
	RedisInterface *redisPackage.RedisInterface
}

type GetHeaderResponse struct {
	Version string                       `json:"version"`
	Data    *capellaAPI.SignedBuilderBid `json:"data"`
}

type ProposerHeaderResponse struct {
	Slot              uint64
	ProposerPubKeyHex string
	Bid               GetHeaderResponse
}

type GetPayloadUtils struct {
	Version              string
	Data                 *capella.ExecutionPayloadHeader
	API                  string
	BuilderWalletAddress string
}

func chunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for {
		if len(slice) == 0 {
			break
		}

		// necessary check to avoid slicing beyond
		// slice capacity
		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}

		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}

	return chunks
}
