package bids

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	"github.com/bsn-eng/pon-wtfpl-relay/bulletinboard"
	"github.com/bsn-eng/pon-wtfpl-relay/redisPackage"
	"github.com/bsn-eng/pon-wtfpl-relay/utils"
	"github.com/sirupsen/logrus"
)

func NewBidBoard(redis redisPackage.RedisInterface, bulletin bulletinboard.RelayMQTT, timeout time.Duration) *BidBoard {
	return &BidBoard{
		redisInterface: redis,
		bulletinBoard:  bulletin,
		bidTimeout:     timeout,
		log: logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "Bid",
		}),
	}
}

func (r *BidBoard) SaveBuilderBid(slot uint64, builderPubkey string, proposerPubKey string, receivedAt time.Time, builderHeader *utils.GetHeaderResponse) (err error) {

	builderBid := &utils.ProposerHeaderResponse{
		Slot:              slot,
		ProposerPubKeyHex: proposerPubKey,
		Bid:               *builderHeader,
	}

	bidKey := fmt.Sprintf("%s-%d", builderKeyBid, slot)
	err = r.redisInterface.HSetObj(bidKey, builderPubkey, builderBid, r.bidTimeout)
	if err != nil {
		return err
	}

	bidTimeKey := fmt.Sprintf("%s-%d", builderTimeKeyBid, slot)
	err = r.redisInterface.Client.HSet(context.Background(), bidTimeKey, builderPubkey, receivedAt.UnixMilli()).Err()
	if err != nil {
		return err
	}
	err = r.redisInterface.Client.Expire(context.Background(), bidTimeKey, r.bidTimeout).Err()
	if err != nil {
		return err
	}

	bidValueKey := fmt.Sprintf("%s-%d", builderValueKeyBid, slot)
	err = r.redisInterface.Client.HSet(context.Background(), bidValueKey, builderPubkey, builderHeader.Data.Message.Value.String()).Err()
	if err != nil {
		return err
	}
	return r.redisInterface.Client.Expire(context.Background(), bidValueKey, r.bidTimeout).Err()
}

func (b *BidBoard) SavePayloadUtils(slot uint64, builder string, blockhash string, payloadUtils *utils.GetPayloadUtils) error {

	bidKey := fmt.Sprintf("%s-%d", bidKeyBuilderUtils, slot)
	builderKey := fmt.Sprintf("%s-%s", builder, blockhash)
	err := b.redisInterface.HSetObj(bidKey, builder, builderKey, b.bidTimeout)
	if err != nil {
		return err
	}
	return nil
}

func (b *BidBoard) PayloadUtils(slot uint64, builder string, blockhash string) (utils.GetPayloadUtils, error) {

	bidKey := fmt.Sprintf("%s-%d", bidKeyBuilderUtils, slot)
	builderKey := fmt.Sprintf("%s-%s", builder, blockhash)
	utilsRelay := new(utils.GetPayloadUtils)
	value, err := b.redisInterface.Client.HGet(context.Background(), bidKey, builderKey).Result()
	if err != nil {
		return utils.GetPayloadUtils{}, err
	}
	err = json.Unmarshal([]byte(value), &utilsRelay)
	return *utilsRelay, err
}

func (r *BidBoard) AuctionBid(slot uint64) (builder string, err error) {

	bidValueKey := fmt.Sprintf("%s-%d", builderValueKeyBid, slot)
	bidValues, err := r.redisInterface.Client.HGetAll(context.Background(), bidValueKey).Result()
	if err != nil {
		return "", err
	}

	topBidValue := uint64(0)
	topBidBuilderPubkey := ""
	for builderPubkey, bidValue := range bidValues {
		bidValueInt, _ := strconv.ParseInt(bidValue, 10, 64)
		if uint64(bidValueInt) > topBidValue {
			topBidValue = uint64(bidValueInt)
			topBidBuilderPubkey = builderPubkey
		}
	}

	if topBidBuilderPubkey == "" {
		return "", errors.New("No Top Bids")
	}

	bidKey := fmt.Sprintf("%s-%d", builderKeyBid, slot)
	bidStr, err := r.redisInterface.Client.HGet(context.Background(), bidKey, topBidBuilderPubkey).Result()
	if err != nil {
		return "", err
	}

	bidHighestKey := fmt.Sprintf("%s-%d", builderHighestKeyBid, slot)
	err = r.redisInterface.Client.Set(context.Background(), bidHighestKey, bidStr, r.bidTimeout).Err()
	if err != nil {
		return "", err
	}

	highestBid := bulletinBoardTypes.RelayHighestBid{
		Slot:             slot,
		BuilderPublicKey: topBidBuilderPubkey,
		Amount:           fmt.Sprintf("%d", topBidValue),
	}
	r.bulletinBoard.HighestBidChannel <- highestBid
	return topBidBuilderPubkey, nil
}

func (b *BidBoard) WinningBid(slot uint64) (*utils.ProposerHeaderResponse, error) {
	b.log.WithFields(logrus.Fields{
		"slot": slot,
	}).Info("Highest Bid Requested")

	bidHighestKey := fmt.Sprintf("%s-%d", builderHighestKeyBid, slot)
	bid := new(utils.ProposerHeaderResponse)
	err := b.redisInterface.GetObj(bidHighestKey, bid)
	if err != nil {
		return nil, err
	}
	return bid, err
}

func (b *BidBoard) GetPayloadDelivered(slot uint64) (value string, err error) {
	b.log.WithFields(logrus.Fields{
		"slot": slot,
	}).Info("Payload Delivered")

	payloadDelivered, err := b.redisInterface.Client.HGet(context.Background(), slotProposerDeliveredKey, fmt.Sprintf("%d", slot)).Result()
	return payloadDelivered, err
}

func (b *BidBoard) PutPayloadDelivered(slot phase0.Slot, builder string) error {
	b.log.WithFields(logrus.Fields{
		"slot": slot,
	}).Info("Payload Delivered")

	err := b.redisInterface.Client.HSet(context.Background(), slotProposerDeliveredKey, fmt.Sprintf("%d", slot), builder).Err()
	return err
}

func (b *BidBoard) BuilderBlockLast(slot uint64, builder string) (value int64, err error) {
	b.log.WithFields(logrus.Fields{
		"slot":    slot,
		"builder": builder,
	}).Info("Last Builder Block Submission")

	bidTimeKey := fmt.Sprintf("%s-%d", builderTimeKeyBid, slot)

	bidBlockBuilder, err := b.redisInterface.Client.HGet(context.Background(), bidTimeKey, builder).Result()
	if err != nil {
		return 0, err
	}
	bid, err := strconv.ParseInt(bidBlockBuilder, 10, 64)
	return bid, err
}
