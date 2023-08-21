package bulletinboard

import (
	"fmt"
	"time"

	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	pahoMQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
)

var (
	relayConnectionHandler pahoMQTT.OnConnectHandler = func(client mqtt.Client) {
		log.Info("MQTT Client Connected For The Relay")
	}

	relayConncetionLostHandler pahoMQTT.ConnectionLostHandler = func(client mqtt.Client, err error) {
		log.Info("Connection Lost To MQTT Client", err)
	}
)

func ClientBrokerUrl(broker string, port uint64) string {
	return fmt.Sprintf("%s://%s:%d", bulletinBoardTypes.TCP, broker, port)
}

func NewMQTTClient(clientParameters bulletinBoardTypes.RelayMQTTOpts, beaconClient *beaconclient.MultiBeaconClient) (*RelayMQTT, error) {

	relayClient := new(RelayMQTT)

	relayClient.Broker = clientParameters.Broker
	relayClient.BeaconInterface = beaconClient

	relayClient.Log = logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
		"package": "BulletinBoard",
		"broker":  clientParameters.Broker,
	})

	relayClient.Channel.HighestBidChannel = make(chan bulletinBoardTypes.RelayHighestBid)
	relayClient.Channel.ProposerHeaderChannel = make(chan bulletinBoardTypes.ProposerHeaderRequest)
	relayClient.Channel.SlotPayloadChannel = make(chan bulletinBoardTypes.SlotPayloadRequest)

	relayClient.ClientOptions = pahoMQTT.NewClientOptions()
	relayClient.ClientOptions.AddBroker(ClientBrokerUrl(clientParameters.Broker, clientParameters.Port))

	relayClient.ClientOptions.SetClientID(clientParameters.ClientID)
	relayClient.ClientOptions.SetUsername(clientParameters.UserName)
	relayClient.ClientOptions.SetPassword(clientParameters.Password)

	relayClient.ClientOptions.OnConnect = relayConnectionHandler
	relayClient.ClientOptions.OnConnectionLost = relayConncetionLostHandler

	relayClient.Client = pahoMQTT.NewClient(relayClient.ClientOptions)

	if relayClientToken := relayClient.Client.Connect(); relayClientToken.Wait() && relayClientToken.Error() != nil {
		relayClient.Log.WithError(relayClientToken.Error()).Fatal("Bulletin Board Client Ready For Relay")
		return nil, relayClientToken.Error()
	}

	relayClient.BulletinBoards()

	relayClient.Log.Info("Bulletin Board Client Ready For Relay")
	return relayClient, nil
}

func (relayClient *RelayMQTT) BulletinBoards() {
	go relayClient.HighestBidPublish()
	go relayClient.SlotHeaderRequested()
	go relayClient.SlotPayloadRequested()
	go relayClient.BountyBidWon()
}

func (relayClient *RelayMQTT) publishBulletinBoard(topic bulletinBoardTypes.MQTTTopic, message string) error {

	relayToken := relayClient.Client.Publish(string(topic), 0, false, message)

	timeout := relayToken.WaitTimeout(time.Duration(mqttTimeout))
	if !timeout {
		return errors.New("Timeout Sending To Broker")
	}

	if relayToken.Error() != nil {
		return relayToken.Error()
	}

	return nil
}
