package bids

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pon-pbs/bbRelay/bulletinboard"
	"github.com/pon-pbs/bbRelay/redisPackage"
)

var (
	builderKeyBid        = "builder-bid"
	builderTimeKeyBid    = "builder-bid-time"
	builderValueKeyBid   = "builder-bid-value"
	builderHighestKeyBid = "builder-highest-bid"
	bidKeyBuilderUtils   = "builder-bid-utils"
)

var (
	slotProposerDeliveredKey = "slot-proposer-payload-delivered"
	slotBountyBidWinnerKey   = "slot-bounty-bid-winner"
)

type BidBoard struct {
	redisInterface redisPackage.RedisInterface
	log            *logrus.Entry
	bulletinBoard  bulletinboard.RelayMQTT
	bidTimeout     time.Duration
}
