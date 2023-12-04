package tcp_routing_test

import (
	"context"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"

	. "github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/routing-acceptance-tests/helpers"
	routing_api "code.cloudfoundry.org/routing-api"
	cfworkflow_helpers "github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
)

func TestTcpRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	routingConfig = helpers.LoadConfig()

	if routingConfig.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = time.Duration(routingConfig.DefaultTimeout) * time.Second
	}

	if routingConfig.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = time.Duration(routingConfig.CfPushTimeout) * time.Second
	}

	RunSpecs(t, "TCP Routing")
}

const preallocatedExternalPorts = 100

var (
	DEFAULT_TIMEOUT          = 2 * time.Minute
	DEFAULT_POLLING_INTERVAL = 5 * time.Second
	CF_PUSH_TIMEOUT          = 2 * time.Minute
	domainName               string

	adminContext     cfworkflow_helpers.UserContext
	routingConfig    helpers.RoutingConfig
	routingApiClient routing_api.Client
	environment      *cfworkflow_helpers.ReproducibleTestSuiteSetup
	logger           lager.Logger
)

var _ = BeforeSuite(func() {
	logger = lagertest.NewTestLogger("test")
	routingApiClient = routing_api.NewClient(routingConfig.RoutingApiUrl, routingConfig.SkipSSLValidation)

	uaaTokenFetcher := helpers.NewTokenFetcher(routingConfig, logger)
	token, err := uaaTokenFetcher.FetchToken(context.Background(), true)
	Expect(err).ToNot(HaveOccurred())

	routingApiClient.SetToken(token.AccessToken)
	_, err = routingApiClient.Routes()
	Expect(err).ToNot(HaveOccurred(), "Routing API is unavailable")

	environment = cfworkflow_helpers.NewTestSuiteSetup(routingConfig.Config)
	adminContext = environment.AdminUserContext()
	regUser := environment.RegularUserContext()
	adminContext.TestSpace = regUser.TestSpace
	adminContext.Org = regUser.Org
	adminContext.Space = regUser.Space
	domainName = routingConfig.TcpAppDomain

	environment.Setup()

	helpers.ValidateRouterGroupName(adminContext, routingConfig.TCPRouterGroup)
})

var _ = AfterSuite(func() {
	environment.Teardown()
	CleanupBuildArtifacts()
})
