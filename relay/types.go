package relay

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

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
	version        string
}

type RelayParams struct {
	DbURL          string
	DatabaseParams databaseTypes.DatabaseOpts
	DbDriver       databaseTypes.DatabaseDriver
	DeleteTables   bool

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

	Version string

	DiscordWebhook string
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

type BuilderWinningBid struct {
	BidID             string  `json:"bid_id"`
	HighestBidValue   big.Int `json:"highest_bid_value"`
	HighestBidBuilder string  `json:"highest_bid_builder"`
}

type RelayConfig struct {
	MQTTBroker string `json:"mqtt_broker"`
	MQTTPort   uint16 `json:"mqtt_port"`
	PublicKey  string `json:"public_key"`
	Chain      uint64 `json:"chain"`
	Slot       uint64 `json:"current_slot"`
}
