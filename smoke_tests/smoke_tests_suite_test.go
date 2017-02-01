package smoke_test

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/routing-acceptance-tests/helpers"
	"code.cloudfoundry.org/routing-api"
	cf_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	cfworkflow_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
)

var (
	DEFAULT_TIMEOUT          = 2 * time.Minute
	DEFAULT_POLLING_INTERVAL = 5 * time.Second
	CF_PUSH_TIMEOUT          = 2 * time.Minute
	routingConfig            helpers.RoutingConfig
	context                  cfworkflow_helpers.SuiteContext
	environment              *cfworkflow_helpers.Environment
)

func TestSmokeTests(t *testing.T) {
	RegisterFailHandler(Fail)
	componentName := "SmokeTests Suite"
	rs := []Reporter{}
	if routingConfig.ArtifactsDirectory != "" {
		cf_helpers.EnableCFTrace(routingConfig.Config, componentName)
		rs = append(rs, cf_helpers.NewJUnitReporter(routingConfig.Config, componentName))
	}
	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)

}

var _ = BeforeSuite(func() {
	routingConfig = helpers.LoadConfig()
	if routingConfig.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = routingConfig.DefaultTimeout * time.Second
	}

	if routingConfig.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = routingConfig.CfPushTimeout * time.Second
	}

	os.Setenv("CF_TRACE", "true")
	context = cfworkflow_helpers.NewContext(routingConfig.Config)
	environment = cfworkflow_helpers.NewEnvironment(context)

	logger := lagertest.NewTestLogger("test")
	routingApiClient := routing_api.NewClient(routingConfig.RoutingApiUrl, routingConfig.SkipSSLValidation)

	uaaClient := helpers.NewUaaClient(routingConfig, logger)
	token, err := uaaClient.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())

	routingApiClient.SetToken(token.AccessToken)
	_, err = routingApiClient.Routes()
	Expect(err).ToNot(HaveOccurred(), "Routing API is unavailable")
})

var _ = AfterSuite(func() {
	environment.Teardown()
	CleanupBuildArtifacts()
})
