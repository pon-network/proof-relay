package cmd

import (
	"fmt"
	"strconv"
	"time"

	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pon-pbs/bbRelay/bls"
	"github.com/pon-pbs/bbRelay/relay"
)

func init() {
	rootCmd.AddCommand(relayCmd)

	relayCmd.Flags().StringVar(&relayURL, "relay-url", relayDefaultURL, "listen address for webserver")
	relayCmd.Flags().StringSliceVar(&beaconNodeURIs, "beacon-uris", defaultBeaconURIs, "beacon endpoints")
	relayCmd.Flags().StringVar(&redisURI, "redis-uri", defaultRedisURI, "redis uri")
	relayCmd.Flags().StringVar(&postgresURL, "db", defaultPostgresURL, "PostgreSQL DSN")
	relayCmd.Flags().StringVar(&apiSecretKey, "secret-key", apiDefaultSecretKey, "secret key for signing bids")
	relayCmd.Flags().StringVar(&network, "network", defaultNetwork, "Which network to use")

	relayCmd.Flags().StringVar(&maxDBConnections, "max-db-connections", maxDBConnectionsDefault, "Maximum DB Connections")
	relayCmd.Flags().StringVar(&maxIdleConnections, "max-idle-connections", maxIdleConnectionsDefault, "Maximum Idle Connections")
	relayCmd.Flags().StringVar(&maxTimeConnection, "max-idle-timeout", maxTimeConnectionDefault, "Maximum Idle Timeout")
	relayCmd.Flags().StringVar(&dbDriver, "db-driver", dbDriverDefault, "Database Driver")

	relayCmd.Flags().StringVar(&ponPoolURL, "pon-pool", ponPoolURLDefault, "Pon Pool URL")
	relayCmd.Flags().StringVar(&ponPoolAPIKey, "pon-pool-API-Key", ponPoolAPIKeyDefault, "Pon Pool API Key")

	relayCmd.Flags().StringVar(&bulletinBoardBroker, "bulletinBoard-broker", bulletinBoardBrokerDefault, "Bulletin Board Broker URL")
	relayCmd.Flags().StringVar(&bulletinBoardPort, "bulletinBoard-port", bulletinBoardPortDefault, "Bulletin Board Port")
	relayCmd.Flags().StringVar(&bulletinBoardClient, "bulletinBoard-client", bulletinBoardClientDefault, "Pon Pool URL")
	relayCmd.Flags().StringVar(&bulletinBoardPassword, "bulletinBoard-password", bulletinBoardPasswordDefault, "Pon Pool URL")

	relayCmd.Flags().StringVar(&reporterURL, "reporter-url", reporterURLDefault, "Reporter Server URL")

	relayCmd.Flags().StringVar(&bidTimeout, "bid-timeout", bidTimeoutDefault, "Bid Timeout")

	relayCmd.Flags().StringVar(&readTimeout, "relay-read-timeout", readTimeoutDefault, "Relay Read Timeout")
	relayCmd.Flags().StringVar(&readHeaderTimeout, "relay-read-header-timeout", readHeaderTimeoutDefault, "Relay Read Header Timeout")
	relayCmd.Flags().StringVar(&writeTimeout, "relay-write-timeout", writeTimeoutDefault, "Relay Write Timeout")
	relayCmd.Flags().StringVar(&idleTimeout, "relay-idle-timeout", idleTimeoutDefault, "Relay Idle Timeout")

	relayCmd.Flags().StringVar(&newRelicApp, "new-relic-application", newRelicAppDefault, "New Relic Application")
	relayCmd.Flags().StringVar(&newRelicLicense, "new-relic-license", newRelicLicenseDefault, "New Relic License")
	relayCmd.Flags().BoolVar(&newRelicForwarding, "new-relic-forwarding", newRelicForwardingDefault, "New Relic Forwarding")

}

var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "Start the Relay",
	Run: func(cmd *cobra.Command, args []string) {
		log := *logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "Relay",
		})

		if len(beaconNodeURIs) == 0 {
			log.Fatal("no beacon endpoints specified")
		}

		if redisURI == "" {
			log.Fatal("No Redis URL Specified")
		}

		if postgresURL == "" {
			log.Fatal("couldn't read db URL")
		}
		maxConnections, _ := strconv.ParseInt(maxDBConnections, 10, 64)
		maxIdle, _ := strconv.ParseInt(maxIdleConnections, 10, 64)
		maxTimeIdle, _ := time.ParseDuration(maxTimeConnection)
		databaseOpts := &databaseTypes.DatabaseOpts{
			MaxConnections:        int(maxConnections),
			MaxIdleConnections:    int(maxIdle),
			MaxIdleTimeConnection: maxTimeIdle,
		}
		if ponPoolURL == "" {
			log.Fatal("couldn't read PON Pool URL")
		}

		bulletinPort, _ := strconv.ParseInt(bulletinBoardPort, 10, 64)
		bulletinBoardParams := &bulletinBoardTypes.RelayMQTTOpts{
			Broker:   bulletinBoardBroker,
			Port:     uint64(bulletinPort),
			ClientID: bulletinBoardClient,
			UserName: bulletinBoardUserName,
			Password: bulletinBoardPassword,
		}

		bid, _ := time.ParseDuration(bidTimeout)

		if apiSecretKey == "" {
			log.Fatal("No secret key specified")
		} else {
			envSkBytes, err := hexutil.Decode(apiSecretKey)
			if err != nil {
				log.WithError(err).Fatal("incorrect secret key provided")
			}
			secretKey, err = bls.SecretKeyFromBytes(envSkBytes[:])
			if err != nil {
				log.WithError(err).Fatal("incorrect builder API secret key provided")
			}
		}

		opts := &relay.RelayParams{
			DbURL:          postgresURL,
			DatabaseParams: *databaseOpts,
			DbDriver:       databaseTypes.DatabaseDriver(dbDriver),

			PonPoolURL:    ponPoolURL,
			PonPoolAPIKey: ponPoolAPIKey,

			BulletinBoardParams: *bulletinBoardParams,

			BeaconClientUrls: beaconNodeURIs,

			ReporterURL: reporterURL,

			URL: relayURL,

			Network: network,

			RedisURI: redisURI,

			BidTimeOut: bid,

			Sk: secretKey,

			NewRelicApp:        newRelicApp,
			NewRelicLicense:    newRelicLicense,
			NewRelicForwarding: newRelicForwarding,
		}

		srv, err := relay.NewRelayAPI(opts, log)
		if err != nil {
			log.WithError(err).Fatal("failed to create service")
		}

		readTimeoutTime, _ := time.ParseDuration(readTimeout)
		readHeaderTimeoutTime, _ := time.ParseDuration(readHeaderTimeout)
		writeTimeoutTime, _ := time.ParseDuration(writeTimeout)
		idleTimeoutTime, _ := time.ParseDuration(idleTimeout)
		serverParams := &relay.RelayServerParams{
			ReadTimeout:       readTimeoutTime,
			ReadHeaderTimeout: readHeaderTimeoutTime,
			WriteTimeout:      writeTimeoutTime,
			IdleTimeout:       idleTimeoutTime,
		}

		fmt.Println(pon_painiting)
		log.Infof("Webserver starting on %s ...", srv.URL)
		err = srv.StartServer(serverParams)
		if err != nil {
			log.WithError(err).Fatal("server error")
		}
	},
}
