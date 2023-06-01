package relay

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	beaconclient "github.com/bsn-eng/pon-wtfpl-relay/beaconinterface"
	bidBoard "github.com/bsn-eng/pon-wtfpl-relay/bids"
	"github.com/bsn-eng/pon-wtfpl-relay/bls"
	"github.com/bsn-eng/pon-wtfpl-relay/bulletinboard"
	"github.com/bsn-eng/pon-wtfpl-relay/database"
	ponpool "github.com/bsn-eng/pon-wtfpl-relay/ponPool"
	"github.com/bsn-eng/pon-wtfpl-relay/reporter"
	"github.com/bsn-eng/pon-wtfpl-relay/signing"
	"github.com/bsn-eng/pon-wtfpl-relay/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

var (
	VersionCapella = "capella"
)

type Signature phase0.BLSSignature
type EcdsaAddress [20]byte
type EcdsaSignature [65]byte
type Hash [32]byte
type PublicKey [48]byte
type Transaction []byte

func (h Hash) String() string {
	return hexutil.Bytes(h[:]).String()
}

func (e EcdsaAddress) String() string {
	return hexutil.Bytes(e[:]).String()
}

func (e EcdsaSignature) String() string {
	return hexutil.Bytes(e[:]).String()
}

func (t Transaction) String() string {
	transaction, _ := json.Marshal(t)
	return string(transaction)
}

type Relay struct {
	db              *database.DatabaseInterface
	ponPool         *ponpool.PonRegistrySubgraph
	bulletinBoard   *bulletinboard.RelayMQTT
	beaconClient    *beaconclient.MultiBeaconClient
	bidBoard        *bidBoard.BidBoard
	URL             string
	blsSk           *bls.SecretKey
	log             *logrus.Entry
	reporterServer  *reporter.ReporterServer
	network         EthNetwork
	publicKey       phase0.BLSPubKey
	client          *http.Client
	server          *http.Server
	relayutils      *utils.RelayUtils
	newRelicApp     *newrelic.Application
	newRelicEnabled bool
}

type RelayParams struct {
	DbURL          string
	DatabaseParams databaseTypes.DatabaseOpts
	DbDriver       databaseTypes.DatabaseDriver

	PonPoolURL    string
	PonPoolAPIKey string

	BulletinBoardParams bulletinBoardTypes.RelayMQTTOpts

	BeaconClientUrls []string

	ReporterURL string

	Network EthNetwork

	RedisURI string

	BidTimeOut time.Duration

	Sk *bls.SecretKey

	NewRelicApp        string
	NewRelicLicense    string
	NewRelicForwarding bool
}

type EthNetwork struct {
	Network             uint64
	GenesisTime         uint64
	DomainBuilder       signing.Domain
	DomainBeaconCapella signing.Domain
}

type RelayServerParams struct {
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type ProposerReqParams struct {
	Slot              uint64
	ProposerPubKeyHex string
	ParentHashHex     string
}

type ExecutionPayload struct {
	ParentHash    phase0.Hash32              `ssz-size:"32"`
	FeeRecipient  bellatrix.ExecutionAddress `ssz-size:"20"`
	StateRoot     [32]byte                   `ssz-size:"32"`
	ReceiptsRoot  [32]byte                   `ssz-size:"32"`
	LogsBloom     [256]byte                  `ssz-size:"256"`
	PrevRandao    [32]byte                   `ssz-size:"32"`
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte                  `ssz-max:"32"`
	BaseFeePerGas [32]byte                `ssz-size:"32"`
	BlockHash     phase0.Hash32           `ssz-size:"32"`
	Transactions  []bellatrix.Transaction `ssz-max:"1048576,1073741824" ssz-size:"?,?"`
	Withdrawals   []*capella.Withdrawal   `ssz-max:"16"`
}

type ProposerPayload struct {
	Version   spec.DataVersion
	Bellatrix *bellatrix.ExecutionPayload
	Capella   *capella.ExecutionPayload
}
