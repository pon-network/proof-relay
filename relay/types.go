package relay

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
	bidBoard "github.com/pon-pbs/bbRelay/bids"
	"github.com/pon-pbs/bbRelay/bls"
	"github.com/pon-pbs/bbRelay/bulletinboard"
	"github.com/pon-pbs/bbRelay/database"
	ponpool "github.com/pon-pbs/bbRelay/ponPool"
	"github.com/pon-pbs/bbRelay/reporter"
	"github.com/pon-pbs/bbRelay/signing"
	"github.com/pon-pbs/bbRelay/utils"
)

var (
	VersionCapella       = "capella"
	EmptyWithdrawalsRoot = phase0.Root([]byte("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"))
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
	db             *database.DatabaseInterface
	ponPool        *ponpool.PonRegistrySubgraph
	bulletinBoard  *bulletinboard.RelayMQTT
	beaconClient   *beaconclient.MultiBeaconClient
	bidBoard       *bidBoard.BidBoard
	URL            string
	blsSk          *bls.SecretKey
	log            *logrus.Entry
	reporterServer *reporter.ReporterServer
	network        EthNetwork
	publicKey      phase0.BLSPubKey
	client         *http.Client
	server         *http.Server
	relayutils     *utils.RelayUtils
}

type RelayParams struct {
	DbURL          string
	DatabaseParams databaseTypes.DatabaseOpts
	DbDriver       databaseTypes.DatabaseDriver

	URL string

	PonPoolURL    string
	PonPoolAPIKey string

	BulletinBoardParams bulletinBoardTypes.RelayMQTTOpts

	BeaconClientUrls []string

	ReporterURL string

	Network string

	RedisURI string

	BidTimeOut time.Duration

	Sk *bls.SecretKey
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
	ParentHash    phase0.Hash32              `ssz-size:"32" json:"parent_hash"`
	FeeRecipient  bellatrix.ExecutionAddress `ssz-size:"20" json:"fee_recipient"`
	StateRoot     [32]byte                   `ssz-size:"32" json:"state_root"`
	ReceiptsRoot  [32]byte                   `ssz-size:"32" json:"receipts_root"`
	LogsBloom     [256]byte                  `ssz-size:"256" json:"logs_bloom"`
	PrevRandao    [32]byte                   `ssz-size:"32" json:"prev_randao"`
	BlockNumber   uint64                     `json:"block_number"`
	GasLimit      uint64                     `json:"gas_limit"`
	GasUsed       uint64                     `json:"gas_used"`
	Timestamp     uint64                     `json:"timestamp"`
	ExtraData     []byte                     `ssz-max:"32" json:"extra_data"`
	BaseFeePerGas [32]byte                   `ssz-size:"32" json:"base_fee_per_gas"`
	BlockHash     phase0.Hash32              `ssz-size:"32" json:"block_hash"`
	Transactions  []bellatrix.Transaction    `ssz-max:"1048576,1073741824" ssz-size:"?,?" json:"transactions" json:"withdrawals"`
	Withdrawals   []*capella.Withdrawal      `ssz-max:"16"`
}

type ProposerPayload struct {
	Version   spec.DataVersion
	Bellatrix *bellatrix.ExecutionPayload
	Capella   *ExecutionPayload
}

type BuilderWinningBid struct {
	BidID             string  `json:"bid_id"`
	HighestBidValue   big.Int `json:"highest_bid_value"`
	HighestBidBuilder string  `json:"highest_bid_builder"`
}
