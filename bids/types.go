package bids

import (
	"time"

	"github.com/bsn-eng/pon-wtfpl-relay/bulletinboard"
	"github.com/bsn-eng/pon-wtfpl-relay/redisPackage"
	"github.com/sirupsen/logrus"
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
)

type BidBoard struct {
	redisInterface redisPackage.RedisInterface
	log            *logrus.Entry
	bulletinBoard  bulletinboard.RelayMQTT
	bidTimeout     time.Duration
}
