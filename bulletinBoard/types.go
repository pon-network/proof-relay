package bulletinboard

import (
	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	pahoMQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

var (
	HighestBidTopic = bulletinBoardTypes.MQTTTopic("topic/HighestBid")
)

type RelayMQTT struct {
	Broker string

	ClientOptions *pahoMQTT.ClientOptions
	Client        pahoMQTT.Client

	Log *logrus.Entry

	HighestBidChannel chan bulletinBoardTypes.RelayHighestBid
}
