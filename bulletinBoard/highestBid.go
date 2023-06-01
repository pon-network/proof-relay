package bulletinboard

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func (relayClient *RelayMQTT) HighestBidPublish() {
	relayClient.Log.Info("Publish Highest Bid Run")
	for {
		select {
		case bid := <-relayClient.HighestBidChannel:

			publishBid := fmt.Sprintf("slot: %d, proposer: %s, amount: %s", bid.Slot, bid.BuilderPublicKey, bid.Amount)

			err := relayClient.publishBulletinBoard(HighestBidTopic, publishBid)
			if err != nil {
				relayClient.Log.WithError(err).Fatalf("Couldn't Update Bid For Proposer %s, Slot %d", bid.BuilderPublicKey, bid.Slot)
				return
			}

			relayClient.Log.WithFields(logrus.Fields{
				"slot":  bid.Slot,
				"value": bid.Amount,
			}).Info("Highest Bid Published")
		}
	}
}
