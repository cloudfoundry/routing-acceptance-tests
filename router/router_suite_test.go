package router

import (
	"encoding/json"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"

	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Suite")
}

var (
	sampleReceiverPath string
	externalIP         string
	routerApiConfig    helpers.RouterApiConfig
	logger             lager.Logger
	routingApiClient   routing_api.Client
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

var externalPort uint32 = 60500

func nextExternalPort() int {
	return int(atomic.AddUint32(&externalPort, 100))
}

var containerPort uint32 = 10500

func nextContainerPort() int {
	return int(atomic.AddUint32(&containerPort, 100))
}

var _ = SynchronizedBeforeSuite(func() []byte {

	cleanupRoutes()

	sampleReceiver, err := gexec.Build("github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/assets/tcp-sample-receiver", "-race")
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

	sampleReceiverPath = context["sample-receiver"]
	externalIP = testutil.GetExternalIP()
	routerApiConfig = helpers.LoadConfig()
	logger = lagertest.NewTestLogger("test")

	routingApiClient = routing_api.NewClient(routerApiConfig.RoutingApiUrl)
	oauth := token_fetcher.OAuthConfig{
		TokenEndpoint: routerApiConfig.OAuth.TokenEndpoint,
		ClientName:    routerApiConfig.OAuth.ClientName,
		ClientSecret:  routerApiConfig.OAuth.ClientSecret,
		Port:          routerApiConfig.OAuth.Port,
	}
	tokenFetcher := token_fetcher.NewTokenFetcher(&oauth)
	token, err := tokenFetcher.FetchToken()
	Expect(err).ToNot(HaveOccurred())
	routingApiClient.SetToken(token.AccessToken)

})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func cleanupRoutes() {
	routerApiConfig := helpers.LoadConfig()
	routingApiClient := routing_api.NewClient(routerApiConfig.RoutingApiUrl)
	oauth := token_fetcher.OAuthConfig{
		TokenEndpoint: routerApiConfig.OAuth.TokenEndpoint,
		ClientName:    routerApiConfig.OAuth.ClientName,
		ClientSecret:  routerApiConfig.OAuth.ClientSecret,
		Port:          routerApiConfig.OAuth.Port,
	}
	tokenFetcher := token_fetcher.NewTokenFetcher(&oauth)
	token, err := tokenFetcher.FetchToken()
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
