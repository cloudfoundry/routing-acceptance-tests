package smoke_test

import (
	"fmt"
	"net/http"

	routing_helpers "code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"code.cloudfoundry.org/routing-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	cfworkflow_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var routerIps []string

var _ = Describe("SmokeTests", func() {

	BeforeEach(func() {
		adminContext = context.AdminUserContext()
		regUser := context.RegularUserContext()
		adminContext.Org = regUser.Org
		adminContext.Space = regUser.Space

		environment.Setup()

		if routingConfig.TcpAppDomain != "" {
			domainName = routingConfig.TcpAppDomain
			cfworkflow_helpers.AsUser(adminContext, context.ShortTimeout(), func() {
				routing_helpers.VerifySharedDomain(routingConfig.TcpAppDomain, DEFAULT_TIMEOUT)
			})
			routerIps = append(routerIps, domainName)
		} else {
			domainName = fmt.Sprintf("%s.%s", generator.PrefixedRandomName("TCP", "DOMAIN"), routingConfig.AppsDomain)

			cfworkflow_helpers.AsUser(adminContext, context.ShortTimeout(), func() {
				routerGroupName := helpers.GetRouterGroupName(context)
				routing_helpers.CreateSharedDomain(domainName, routerGroupName, DEFAULT_TIMEOUT)
				routing_helpers.VerifySharedDomain(domainName, DEFAULT_TIMEOUT)
			})
			routerIps = routingConfig.Addresses
		}
		appName = routing_helpers.GenerateAppName()
		helpers.UpdateOrgQuota(context)
	})

	AfterEach(func() {
		routing_helpers.DeleteApp(appName, DEFAULT_TIMEOUT)
		if routingConfig.TcpAppDomain == "" {
			routing_helpers.DeleteSharedDomain(domainName, DEFAULT_TIMEOUT)
		}
	})

	It("map tcp route to app successfully ", func() {
		routing_helpers.PushAppNoStart(appName, tcpSampleGolang, routingConfig.GoBuildpackName, domainName, CF_PUSH_TIMEOUT, "256M", "--no-route")
		routing_helpers.EnableDiego(appName, DEFAULT_TIMEOUT)
		routing_helpers.MapRandomTcpRouteToApp(appName, domainName, DEFAULT_TIMEOUT)
		routing_helpers.StartApp(appName, DEFAULT_TIMEOUT)
		port := routing_helpers.GetPortFromAppsInfo(appName, domainName, DEFAULT_TIMEOUT)

		// check tcp route is reachable from list of all Addresses
		for _, routingAddr := range routerIps {
			curlAppSuccess(routingAddr, port)
		}
		// delete the route and verify route is not reachable from all Addresses
		routing_helpers.DeleteTcpRoute(domainName, port, DEFAULT_TIMEOUT)

		for _, routingAddr := range routerIps {
			curlAppFailure(routingAddr, port)
		}
	})
})

func curlAppSuccess(domainName, port string) {
	appUrl := fmt.Sprintf("http://%s:%s", domainName, port)
	resp, err := http.Get(appUrl)
	Expect(err).NotTo(HaveOccurred())

	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func curlAppFailure(domainName, port string) {
	appUrl := fmt.Sprintf("http://%s:%s", domainName, port)

	_, err := http.Get(appUrl)
	Expect(err).To(HaveOccurred())
}
