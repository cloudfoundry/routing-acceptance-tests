package router

import (
	"encoding/json"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"

	"github.com/cloudfoundry-incubator/cf-router-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Suite")
}

const preallocatedExternalPorts = 100

var (
	sampleReceiverPath string
	externalIP         string
	routerApiConfig    helpers.RouterApiConfig
	logger             lager.Logger
	routingApiClient   routing_api.Client
	externalPort       uint32
	bucketSize         int
	containerPort      uint32
)

func validateTcpRouteMapping(tcpRouteMapping db.TcpRouteMapping) bool {
	if tcpRouteMapping.TcpRoute.RouterGroupGuid == "" {
		return false
	}

	if tcpRouteMapping.TcpRoute.ExternalPort <= 0 {
		return false
	}

	if tcpRouteMapping.HostIP == "" {
		return false
	}

	if tcpRouteMapping.HostPort <= 0 {
		return false
	}

	return true
}

func nextExternalPort() int {
	port := int(atomic.AddUint32(&externalPort, uint32(1))) + (GinkgoParallelNode()-1)*bucketSize
	logger.Info("next-external-port", lager.Data{"ginkgo-parallel-node": GinkgoParallelNode(), "externalPort": port})
	return port
}

func nextContainerPort() int {
	port := int(atomic.AddUint32(&containerPort, uint32(1))) + (GinkgoParallelNode()-1)*bucketSize
	logger.Info("next-container-port", lager.Data{"ginkgo-parallel-node": GinkgoParallelNode(), "containerPort": port})
	return port
}

var _ = SynchronizedBeforeSuite(func() []byte {
	cleanupRoutes(lagertest.NewTestLogger("cleanup"))

	sampleReceiver, err := gexec.Build("github.com/cloudfoundry-incubator/cf-router-acceptance-tests/assets/tcp-sample-receiver", "-race")
	Expect(err).NotTo(HaveOccurred())
	payload, err := json.Marshal(map[string]string{
		"sample-receiver": sampleReceiver,
	})

	Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	context := map[string]string{}

	err := json.Unmarshal(payload, &context)
	Expect(err).NotTo(HaveOccurred())
	logger = lagertest.NewTestLogger("test")
	sampleReceiverPath = context["sample-receiver"]
	externalIP = testutil.GetExternalIP()
	routerApiConfig = helpers.LoadConfig()

	routingApiClient = routing_api.NewClient(routerApiConfig.RoutingApiUrl)

	tokenFetcher, err := createTokenFetcher(logger, routerApiConfig)
	Expect(err).ToNot(HaveOccurred())

	token, err := tokenFetcher.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())
	routingApiClient.SetToken(token.AccessToken)
	externalPort = 59999
	containerPort = 5000
	bucketSize = preallocatedExternalPorts / config.GinkgoConfig.ParallelTotal
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func cleanupRoutes(logger lager.Logger) {
	routerApiConfig := helpers.LoadConfig()
	routingApiClient := routing_api.NewClient(routerApiConfig.RoutingApiUrl)

	tokenFetcher, err := createTokenFetcher(logger, routerApiConfig)
	Expect(err).ToNot(HaveOccurred())

	token, err := tokenFetcher.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())
	routingApiClient.SetToken(token.AccessToken)

	// Cleaning up all the pre-existing routes.
	tcpRouteMappings, err := routingApiClient.TcpRouteMappings()
	Expect(err).ToNot(HaveOccurred())
	deleteTcpRouteMappings := make([]db.TcpRouteMapping, 0)
	for _, tcpRouteMapping := range tcpRouteMappings {
		if validateTcpRouteMapping(tcpRouteMapping) {
			deleteTcpRouteMappings = append(deleteTcpRouteMappings, tcpRouteMapping)
		}
	}
	err = routingApiClient.DeleteTcpRouteMappings(deleteTcpRouteMappings)
	Expect(err).ToNot(HaveOccurred())
}

func createTokenFetcher(logger lager.Logger, routerApiConfig helpers.RouterApiConfig) (token_fetcher.TokenFetcher, error) {
	oauth := token_fetcher.OAuthConfig{
		TokenEndpoint: routerApiConfig.OAuth.TokenEndpoint,
		ClientName:    routerApiConfig.OAuth.ClientName,
		ClientSecret:  routerApiConfig.OAuth.ClientSecret,
		Port:          routerApiConfig.OAuth.Port,
	}
	clock := clock.NewClock()

	logger.Debug("creating-uaa-token-fetcher")

	tokenFetcherConfig := token_fetcher.TokenFetcherConfig{
		MaxNumberOfRetries:   uint32(3),
		RetryInterval:        5 * time.Second,
		ExpirationBufferTime: int64(30),
	}
	return token_fetcher.NewTokenFetcher(logger, &oauth, tokenFetcherConfig, clock)
}
