package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/alphagov/paas-elasticache-broker/providers/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pivotal-cf/brokerapi"
)

var (
	configFilePath string
	port           string

	logLevels = map[string]lager.LogLevel{
		"DEBUG": lager.DEBUG,
		"INFO":  lager.INFO,
		"ERROR": lager.ERROR,
		"FATAL": lager.FATAL,
	}
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "Location of the config file")
	flag.StringVar(&port, "port", "3000", "Listen port")
}

func newLogger(logLevel string) lager.Logger {
	laggerLogLevel, ok := logLevels[strings.ToUpper(logLevel)]
	if !ok {
		log.Fatal("Invalid log level: ", logLevel)
	}

	logger := lager.NewLogger("elasticache-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, laggerLogLevel))

	return logger
}

func newBroker(config broker.Config, logger lager.Logger) (*broker.Broker, error) {
	awsConfig := aws.NewConfig().WithRegion(config.Region)
	awsSession := session.Must(session.NewSession(awsConfig))
	svc := elasticache.New(awsSession)

	awsAccountID, err := userAccount(sts.New(awsSession))
	if err != nil {
		return nil, err
	}
	awsPartition := "aws"
	awsRegion := config.Region

	return broker.New(config, redis.NewProvider(svc, awsAccountID, awsPartition, awsRegion, logger, config.AuthTokenSeed), logger), nil
}

func userAccount(stssvc *sts.STS) (string, error) {
	getCallerIdentityInput := &sts.GetCallerIdentityInput{}
	getCallerIdentityOutput, err := stssvc.GetCallerIdentity(getCallerIdentityInput)
	if err != nil {
		return "", err
	}
	return *getCallerIdentityOutput.Account, nil
}

func newHTTPHandler(serviceBroker *broker.Broker, logger lager.Logger, config broker.Config) http.Handler {
	credentials := brokerapi.BrokerCredentials{
		Username: config.Username,
		Password: config.Password,
	}

	brokerAPI := brokerapi.New(serviceBroker, logger, credentials)
	mux := http.NewServeMux()
	mux.Handle("/", brokerAPI)
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func main() {
	flag.Parse()

	config, err := broker.LoadConfig(configFilePath)
	if err != nil {
		log.Fatalf("Error loading config file: %s", err)
	}

	logger := newLogger(config.LogLevel)

	serviceBroker, err := newBroker(config, logger)
	if err != nil {
		log.Fatalf("Error creating broker: %s", err)
	}

	server := newHTTPHandler(serviceBroker, logger, config)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Error listening to port %s: %s", port, err)
	}
	fmt.Println("ElastiCache Service Broker started on port " + port + "...")
	http.Serve(listener, server)
}
