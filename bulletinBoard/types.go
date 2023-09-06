package bulletinboard

import (
	"time"

	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	pahoMQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
)

var (
	mqttTimeout = time.Millisecond
)

var (
	HighestBidTopic             = bulletinBoardTypes.MQTTTopic("topic/HighestBid")
	ProposerRequestTopic        = bulletinBoardTypes.MQTTTopic("topic/ProposerSlotHeaderRequest")
	ProposerPayloadRequestTopic = bulletinBoardTypes.MQTTTopic("topic/ProposerPayloadRequest")
	BountyBidTopic              = bulletinBoardTypes.MQTTTopic("topic/BountyBidWon")
)

type RelayMQTTChannels struct {
	HighestBidChannel     chan bulletinBoardTypes.RelayHighestBid
	ProposerHeaderChannel chan bulletinBoardTypes.ProposerHeaderRequest
	SlotPayloadChannel    chan bulletinBoardTypes.SlotPayloadRequest
	BountyBidChannel      chan bulletinBoardTypes.BountyBidWon
}

type RelayMQTT struct {
	Broker        string
	Port          uint64
	ClientOptions *pahoMQTT.ClientOptions
	Client        pahoMQTT.Client

	BeaconInterface *beaconclient.MultiBeaconClient

	Log *logrus.Entry

	Channel RelayMQTTChannels
}
