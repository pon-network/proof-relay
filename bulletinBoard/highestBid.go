package bulletinboard

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func (relayClient *RelayMQTT) HighestBidPublish() {

	for {
		select {
		case bid := <-relayClient.Channel.HighestBidChannel:
			relayClient.Log.Info("Publish Highest Bid Run")
			publishBid := fmt.Sprintf("slot: %d, builder: %s, amount: %s", bid.Slot, bid.BuilderPublicKey, bid.Amount)

			err := relayClient.publishBulletinBoard(HighestBidTopic, publishBid)
			if err != nil {
				relayClient.Log.WithError(err).Errorf("Couldn't Update Bid For Proposer %s, Slot %d", bid.BuilderPublicKey, bid.Slot)
			} else {
				relayClient.Log.WithFields(logrus.Fields{
					"slot":  bid.Slot,
					"value": bid.Amount,
				}).Info("Highest Bid Published")
			}
		}
	}
}

func (relayClient *RelayMQTT) SlotHeaderRequested() {

	for {
		select {
		case slot := <-relayClient.Channel.ProposerHeaderChannel:
			relayClient.Log.Info("Proposer Header Request Run")
			proposerRequest := fmt.Sprintf("slot: %d, proposer: %s, timestamp: %d", slot.Slot, slot.Proposer, slot.Timestamp)

			err := relayClient.publishBulletinBoard(ProposerRequestTopic, proposerRequest)
			if err != nil {
				relayClient.Log.WithError(err).Errorf("Couldn't Update Proposer Request For Proposer %s, Slot %d", slot.Proposer, slot.Slot)
			} else {
				relayClient.Log.WithFields(logrus.Fields{
					"slot":     slot.Slot,
					"proposer": slot.Proposer,
				}).Info("Proposer Slot Request")
			}
		}
	}
}

func (relayClient *RelayMQTT) SlotPayloadRequested() {

	for {
		select {
		case slot := <-relayClient.Channel.SlotPayloadChannel:
			relayClient.Log.Info("Proposer Payload Request Run")
			proposerRequest := fmt.Sprintf("slot: %d, proposer: %s", slot.Slot, slot.Proposer)

			err := relayClient.publishBulletinBoard(ProposerPayloadRequestTopic, proposerRequest)
			if err != nil {
				relayClient.Log.WithError(err).Errorf("Couldn't Update Proposer Request For Proposer %s, Slot %d", slot.Proposer, slot.Slot)
			} else {
				relayClient.Log.WithFields(logrus.Fields{
					"slot":     slot.Slot,
					"proposer": slot.Proposer,
				}).Info("Slot Payload Requested")
			}
		}
	}
}

func (relayClient *RelayMQTT) BountyBidWon() {

	for {
		select {
		case slot := <-relayClient.Channel.BountyBidChannel:
			relayClient.Log.Info("Proposer Payload Request Run")
			builderBountyBid := fmt.Sprintf("slot: %d, builder: %s", slot.Slot, slot.Builder)

			err := relayClient.publishBulletinBoard(BountyBidTopic, builderBountyBid)
			if err != nil {
				relayClient.Log.WithError(err).Errorf("Couldn't Update Bounty Bid For Builder %s, Slot %d", slot.Builder, slot.Slot)
			} else {
				relayClient.Log.WithFields(logrus.Fields{
					"slot":    slot.Slot,
					"builder": slot.Builder,
				}).Info("Bounty Bid Won")
			}
		}
	}
}
