package smoke_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	routing_helpers "code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"code.cloudfoundry.org/routing-acceptance-tests/helpers"
	"code.cloudfoundry.org/routing-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	cfworkflow_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var routerIps []string
var (
	appName                 string
	domainName              string
	tcpSampleGolang         = assets.NewAssets().TcpSampleGolang
	adminContext            cfworkflow_helpers.UserContext
	DEFAULT_RW_TIMEOUT      = 2 * time.Second
	DEFAULT_CONNECT_TIMEOUT = 2 * time.Second
	regUser                 cfworkflow_helpers.UserContext
)

var _ = Describe("SmokeTests", func() {

	BeforeEach(func() {
		if routingConfig.TcpAppDomain != "" {
			domainName = routingConfig.TcpAppDomain
			cfworkflow_helpers.AsUser(adminContext, adminContext.Timeout, func() {
				routing_helpers.VerifySharedDomain(routingConfig.TcpAppDomain, DEFAULT_TIMEOUT)
			})
			routerIps = append(routerIps, domainName)
		} else {
			domainName = fmt.Sprintf("%s.%s", generator.PrefixedRandomName("TCP", "DOMAIN"), routingConfig.AppsDomain)

			cfworkflow_helpers.AsUser(adminContext, adminContext.Timeout, func() {
				routing_helpers.CreateSharedDomain(domainName, routingConfig.TCPRouterGroup, DEFAULT_TIMEOUT)
				routing_helpers.VerifySharedDomain(domainName, DEFAULT_TIMEOUT)
			})
			routerIps = routingConfig.Addresses
		}
		appName = routing_helpers.GenerateAppName()
		helpers.UpdateOrgQuota(adminContext)
	})

	AfterEach(func() {
		routing_helpers.AppReport(appName, DEFAULT_TIMEOUT)
		routing_helpers.DeleteApp(appName, DEFAULT_TIMEOUT)
		if routingConfig.TcpAppDomain == "" {
			cfworkflow_helpers.AsUser(adminContext, adminContext.Timeout, func() {
				routing_helpers.DeleteSharedDomain(domainName, DEFAULT_TIMEOUT)
			})
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
	fmt.Fprintf(os.Stdout, "\nConnecting to URL %s... \n", appUrl)
	resp, err := http.Get(appUrl)
	Expect(err).NotTo(HaveOccurred())
	fmt.Fprintf(os.Stdout, "\nReceived response %d\n", resp.StatusCode)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func curlAppFailure(domainName, port string) {
	appUrl := fmt.Sprintf("%s:%s", domainName, port)
	fmt.Fprintf(os.Stdout, "\nConnecting to URL %s... \n", appUrl)
	conn, err := net.DialTimeout("tcp", appUrl, DEFAULT_CONNECT_TIMEOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nReceived error while connecting %s\n", err.Error())
		return
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	err = conn.SetWriteDeadline(time.Now().Add(DEFAULT_RW_TIMEOUT))
	testBytes := []byte("GET / HTTP/1.1 \n\n")
	_, err = conn.Write(testBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nReceived error while writing to connection %s\n", err.Error())
		return
	}
	err = conn.SetReadDeadline(time.Now().Add(DEFAULT_RW_TIMEOUT))
	readBytes := make([]byte, 1024)
	_, err = conn.Read(readBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nReceived error while reading from connection %s\n", err.Error())
	}
	Expect(err).To(HaveOccurred())
	return
}
