package bids

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	"github.com/go-redis/redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/pon-pbs/bbRelay/bulletinboard"
	"github.com/pon-pbs/bbRelay/redisPackage"
	"github.com/pon-pbs/bbRelay/utils"
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

func (r *BidBoard) SaveBuilderBid(slot uint64, builderPubkey string, proposerPubKey string, receivedAt uint64, builderHeader *utils.GetHeaderResponse) (err error) {

	builderBid := &utils.ProposerHeaderResponse{
		Slot:              slot,
		ProposerPubKeyHex: proposerPubKey,
		Bid:               *builderHeader,
	}

	/*
		It is used to keep the bid as json

			Bid Key-
			builderKeyBid-slot
								|--builderPubkey1 =
								|--builderPubkey2 =
								|--builderPubkey3 =
	*/

	bidKey := fmt.Sprintf("%s-%d", builderKeyBid, slot)
	err = r.redisInterface.HSetObj(bidKey, builderPubkey, builderBid, r.bidTimeout)
	if err != nil {
		return err
	}

	/*
		It is used to make sure that when builder sends a new block, we use the latest one only

			Bid Time Key-
			builderTimeKeyBid-slot
								|--builderPubkey1 =
								|--builderPubkey2 =
								|--builderPubkey3 =
	*/

	bidTimeKey := fmt.Sprintf("%s-%d", builderTimeKeyBid, slot)
	err = r.redisInterface.Client.HSet(context.Background(), bidTimeKey, builderPubkey, receivedAt).Err()
	if err != nil {
		return err
	}
	err = r.redisInterface.Client.Expire(context.Background(), bidTimeKey, r.bidTimeout).Err()
	if err != nil {
		return err
	}

	/*
		It is used to store value of latest bid of a builder for a slot. It is used in auction

			Bid Value Key-
			builderValueKeyBid-slot
								|--builderPubkey1 =
								|--builderPubkey2 =
								|--builderPubkey3 =
	*/

	bidValueKey := fmt.Sprintf("%s-%d", builderValueKeyBid, slot)
	err = r.redisInterface.Client.HSet(context.Background(), bidValueKey, builderPubkey, fmt.Sprintf("%d", builderHeader.Data.Message.Value)).Err()
	if err != nil {
		return err
	}
	return r.redisInterface.Client.Expire(context.Background(), bidValueKey, r.bidTimeout).Err()
}

func (b *BidBoard) SavePayloadUtils(slot uint64, proposer string, blockhash string, payloadUtils *utils.GetPayloadUtils) error {

	/*
		It is used to keep the builder API for bids (Blockhash) which is used in get payload

			Bid Key-
			bidKeyBuilderUtils-slot
								|--blockhash =
								|--blockhash =
								|--blockhash =
	*/

	bidKey := fmt.Sprintf("%s-%d", bidKeyBuilderUtils, slot)
	err := b.redisInterface.HSetObj(bidKey, blockhash, payloadUtils, b.bidTimeout)
	if err != nil {
		return err
	}
	return nil
}

func (b *BidBoard) PayloadUtils(slot uint64, blockhash string) (utils.GetPayloadUtils, error) {

	bidKey := fmt.Sprintf("%s-%d", bidKeyBuilderUtils, slot)
	utilsRelay := new(utils.GetPayloadUtils)
	value, err := b.redisInterface.Client.HGet(context.Background(), bidKey, blockhash).Result()
	if err != nil {
		return utils.GetPayloadUtils{}, err
	}
	err = json.Unmarshal([]byte(value), &utilsRelay)
	return *utilsRelay, err
}

func (b *BidBoard) AuctionBid(slot uint64) (builder string, value uint64, err error) {

	b.log.WithFields(logrus.Fields{
		"slot": slot,
	}).Info("Auction Requested By Relay")

	bidValueKey := fmt.Sprintf("%s-%d", builderValueKeyBid, slot)
	bidValues, err := b.redisInterface.Client.HGetAll(context.Background(), bidValueKey).Result()
	if err != nil {
		return "", 0, err
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
		return "", 0, errors.New(fmt.Sprintf("No Bids For Slot %d, Auction Not Possible For Slot", slot))
	}

	bidKey := fmt.Sprintf("%s-%d", builderKeyBid, slot)
	bidStr, err := b.redisInterface.Client.HGet(context.Background(), bidKey, topBidBuilderPubkey).Result()
	if err != nil {
		return "", 0, err
	}

	bidHighestKey := fmt.Sprintf("%s-%d", builderHighestKeyBid, slot)
	err = b.redisInterface.Client.Set(context.Background(), bidHighestKey, bidStr, b.bidTimeout).Err()
	if err != nil {
		return "", 0, err
	}

	highestBid := bulletinBoardTypes.RelayHighestBid{
		Slot:             slot,
		BuilderPublicKey: topBidBuilderPubkey,
		Amount:           fmt.Sprintf("%d", topBidValue),
	}
	b.bulletinBoard.Channel.HighestBidChannel <- highestBid

	return topBidBuilderPubkey, topBidValue, nil
}

func (b *BidBoard) WinningBid(slot uint64) (*utils.ProposerHeaderResponse, error) {
	b.log.WithFields(logrus.Fields{
		"slot": slot,
	}).Info("Winning Bid Requested By Relay")

	bidHighestKey := fmt.Sprintf("%s-%d", builderHighestKeyBid, slot)
	bid := new(utils.ProposerHeaderResponse)
	err := b.redisInterface.GetObj(bidHighestKey, bid)
	if err != nil {
		return nil, err
	}
	return bid, err
}

func (b *BidBoard) GetPayloadDelivered(slot uint64) (value string, err error) {

	payloadDelivered, err := b.redisInterface.Client.HGet(context.Background(), slotProposerDeliveredKey, fmt.Sprintf("%d", slot)).Result()
	return payloadDelivered, err
}

func (b *BidBoard) PutPayloadDelivered(slot uint64, builder string) error {

	b.log.WithFields(logrus.Fields{
		"slot": slot,
	}).Info("Putting Payload Delivered Epoch")

	err := b.redisInterface.Client.HSet(context.Background(), slotProposerDeliveredKey, fmt.Sprintf("%d", slot), builder).Err()
	return err
}

func (b *BidBoard) BuilderBlockLast(slot uint64, builder string) (value int64, err error) {

	b.log.WithFields(logrus.Fields{
		"slot":    slot,
		"builder": builder,
	}).Info("Getting Builder's Last Block Submission Epoch")

	/*
		Bid Time Key-
		builderTimeKeyBid-slot
							|--Builder1 =
							|--Builder2 =
							|--Builder3 =
	*/

	bidTimeKey := fmt.Sprintf("%s-%d", builderTimeKeyBid, slot)

	bidBlockBuilder, err := b.redisInterface.Client.HGet(context.Background(), bidTimeKey, builder).Result()
	if err != nil {
		return 0, err
	}
	bidEpoch, err := strconv.ParseInt(bidBlockBuilder, 10, 64)
	return bidEpoch, err
}

///////////////////////////////////////////////////////////////////////////
//                         Bounty Bid Functions
///////////////////////////////////////////////////////////////////////////

// @dev Gives Winner Of Bounty Bid Of A Slot
func (b *BidBoard) GetBountyBidForSlot(slot uint64) (builder string, err error) {

	bountyBidWinner, err := b.redisInterface.Client.HGet(context.Background(), slotBountyBidWinnerKey, fmt.Sprintf("%d", slot)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return bountyBidWinner, err
}

// @dev Gives highest bid for open bidding
func (b *BidBoard) GetOpenAuctionHighestBid(slot uint64) (value uint64, err error) {

	winningBid, err := b.WinningBid(slot)
	if err != nil {
		/// @dev If no bid is available, send 0
		if err == redis.Nil {
			return 0, nil
		} else {
			return 0, err
		}
	}
	return winningBid.Bid.Data.Message.Value.Uint64(), nil
}

// @dev Sets The Bounty Bid Winner
func (b *BidBoard) SetBountyBidForSlot(slot uint64, builder string) (bountyBidWin bool, err error) {

	bountyBidWinner, err := b.GetBountyBidForSlot(slot)
	if err != nil {
		return false, err
	}

	if bountyBidWinner != "" {
		return false, nil
	}

	/*
		Bid Time Key-
		slotBountyBidWinnerKey
							|--Slot1 =
							|--Slot2 =
							|--Slot3 =
	*/

	err = b.redisInterface.Client.HSet(context.Background(), slotBountyBidWinnerKey, fmt.Sprintf("%d", slot), builder).Err()
	if err != nil {
		return false, nil
	}

	b.log.WithFields(logrus.Fields{
		"slot":    slot,
		"builder": builder,
	}).Info("Bounty Bid Won")

	bountyBid := bulletinBoardTypes.BountyBidWon{
		Slot:    slot,
		Builder: builder,
	}
	b.bulletinBoard.Channel.BountyBidChannel <- bountyBid

	return true, nil
}
