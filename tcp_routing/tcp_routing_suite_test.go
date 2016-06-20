package tcp_routing_test

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"

	routing_helpers "github.com/cloudfoundry-incubator/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	cf_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/routing-acceptance-tests/helpers"
	routing_api "github.com/cloudfoundry-incubator/routing-api"
	uaaclient "github.com/cloudfoundry-incubator/uaa-go-client"
	uaaconfig "github.com/cloudfoundry-incubator/uaa-go-client/config"
)

func TestTcpRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	routingConfig = helpers.LoadConfig()
	if routingConfig.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = routingConfig.DefaultTimeout * time.Second
	}

	if routingConfig.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = routingConfig.CfPushTimeout * time.Second
	}

	componentName := "TCP Routing"

	rs := []Reporter{}

	context = cf_helpers.NewContext(routingConfig.Config)
	environment = cf_helpers.NewEnvironment(context)

	if routingConfig.ArtifactsDirectory != "" {
		cf_helpers.EnableCFTrace(routingConfig.Config, componentName)
		rs = append(rs, cf_helpers.NewJUnitReporter(routingConfig.Config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

const preallocatedExternalPorts = 100

var (
	DEFAULT_TIMEOUT          = 2 * time.Minute
	DEFAULT_POLLING_INTERVAL = 5 * time.Second
	CF_PUSH_TIMEOUT          = 2 * time.Minute
	domainName               string

	routingConfig    helpers.RoutingConfig
	routingApiClient routing_api.Client
	context          cf_helpers.SuiteContext
	environment      *cf_helpers.Environment
	logger           lager.Logger
)

func checkRoutingApiEnabled() {
	cmd := cf.Cf("curl", "/v2/info")
	Expect(cmd.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	type infoResponse struct {
		RoutingEndpoint string `json:"routing_endpoint"`
	}

	var response infoResponse
	err := json.Unmarshal(cmd.Buffer().Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	enabled := response.RoutingEndpoint != ""
	Expect(enabled).To(BeTrue(), "Routing API is not enabled")
}

var _ = BeforeSuite(func() {
	logger = lagertest.NewTestLogger("test")
	routingApiClient = routing_api.NewClient(routingConfig.RoutingApiUrl)

	uaaClient := newUaaClient(routingConfig, logger)
	token, err := uaaClient.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())

	routingApiClient.SetToken(token.AccessToken)
	routerGroupGuid := getRouterGroupGuid(routingApiClient)
	domainName = fmt.Sprintf("%s.%s", generator.PrefixedRandomName("TCP-DOMAIN-"), routingConfig.AppsDomain)
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		checkRoutingApiEnabled()
		routing_helpers.CreateSharedDomain(domainName, routerGroupGuid, DEFAULT_TIMEOUT)
		Expect(routing_helpers.GetDomainGuid(domainName, DEFAULT_TIMEOUT)).NotTo(BeEmpty())
	})

	environment.Setup()
})

var _ = AfterSuite(func() {
	environment.Teardown()
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		routing_helpers.DeleteSharedDomain(domainName, DEFAULT_TIMEOUT)
	})
	CleanupBuildArtifacts()
})

func newUaaClient(routerApiConfig helpers.RoutingConfig, logger lager.Logger) uaaclient.Client {

	tokenURL := fmt.Sprintf("%s:%d", routerApiConfig.OAuth.TokenEndpoint, routerApiConfig.OAuth.Port)

	cfg := &uaaconfig.Config{
		UaaEndpoint:           tokenURL,
		SkipVerification:      routerApiConfig.OAuth.SkipSSLValidation,
		ClientName:            routerApiConfig.OAuth.ClientName,
		ClientSecret:          routerApiConfig.OAuth.ClientSecret,
		MaxNumberOfRetries:    3,
		RetryInterval:         500 * time.Millisecond,
		ExpirationBufferInSec: 30,
	}

	uaaClient, err := uaaclient.NewClient(logger, cfg, clock.NewClock())
	Expect(err).ToNot(HaveOccurred())

	_, err = uaaClient.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())

	return uaaClient
}

func getRouterGroupGuid(routingApiClient routing_api.Client) string {
	routerGroups, err := routingApiClient.RouterGroups()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(routerGroups)).ToNot(Equal(0))
	return routerGroups[0].Guid
}
