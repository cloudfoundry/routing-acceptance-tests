package routing_api

import (
	"os/exec"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/helpers"

	"testing"
)

func Rtr(args ...string) *Session {
	portString := strconv.Itoa(routerApiConfig.OAuth.Port)
	oauthUrl := routerApiConfig.OAuth.TokenEndpoint + ":" + portString
	args = append(args, "--api", routerApiConfig.RoutingApiUrl, "--client-id", routerApiConfig.OAuth.ClientName, "--client-secret", routerApiConfig.OAuth.ClientSecret, "--oauth-url", oauthUrl)
	if routerApiConfig.OAuth.SkipOAuthTLSVerification {
		args = append(args, "--skip-tls-verification")
	}
	session, err := Start(exec.Command("rtr", args...), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}

const (
	DEFAULT_TIMEOUT      = 30 * time.Second
	CF_PUSH_TIMEOUT      = 2 * time.Minute
	DEFAULT_MEMORY_LIMIT = "256M"
)

var routerApiConfig helpers.RoutingConfig

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	routerApiConfig = helpers.LoadConfig()

	BeforeSuite(func() {
		Expect(routerApiConfig.OAuth.ClientSecret).ToNot(Equal(""), "Must provide a client secret for the routing suite")
		Expect(routerApiConfig.RoutingApiUrl).ToNot(Equal(""), "Must provide either routing_api_url or system_domain for the routing suite")
	})

	RunSpecs(t, "Routing API Suite")
}
