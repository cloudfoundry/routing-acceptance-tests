package routing_api

import (
	"net/url"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-router-acceptance-tests/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Registration", func() {
	var (
		oauthPassword      string
		oauthUrl           string
		routingApiEndpoint string

		route     string
		routeJSON string
	)

	BeforeEach(func() {
		oauthPassword = routerApiConfig.OAuth.ClientSecret
		oauthUrl = routerApiConfig.OAuth.TokenEndpoint
		routingApiEndpoint = routerApiConfig.RoutingApiUrl
	})

	Describe("HTTP Route", func() {
		var (
			eventsSession *Session
		)
		BeforeEach(func() {
			route = helpers.RandomName()
			routeJSON = `[{"route":"` + route + `","port":65340,"ip":"1.2.3.4","ttl":60}]`
		})

		AfterEach(func() {
			if eventsSession != nil {
				eventsSession.Kill()
			}
		})

		It("can register, list, subscribe to sse and unregister routes", func() {
			args := []string{"events", "--http", "--api", routingApiEndpoint, "--client-id", "tcp_emitter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			eventsSession = Rtr(args...)
			routingHostPort, err := url.Parse(routingApiEndpoint)
			urlParts := strings.Split(routingHostPort.Host, ":")
			Expect(err).ToNot(HaveOccurred())
			Expect(urlParts).To(HaveLen(2))
			expectedPort := "\"" + "port" + "\"" + ":" + urlParts[1]
			Eventually(eventsSession.Out, 70*time.Second).Should(Say(expectedPort))

			args = []string{"register", routeJSON, "--api", routingApiEndpoint, "--client-id", "tcp_emitter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session := Rtr(args...)
			Eventually(session.Out).Should(Say("Successfully registered routes"))
			Eventually(eventsSession.Out, 10*time.Second).Should(Say(route))
			Eventually(eventsSession.Out).Should(Say(`"port":65340`))
			Eventually(eventsSession.Out).Should(Say(`"ip":"1.2.3.4"`))
			Eventually(eventsSession.Out).Should(Say(`"ttl":60`))
			Eventually(eventsSession.Out).Should(Say(`"Action":"Upsert"`))

			args = []string{"list", "--api", routingApiEndpoint, "--client-id", "tcp_emitter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say(route))

			args = []string{"unregister", routeJSON, "--api", routingApiEndpoint, "--client-id", "tcp_emitter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say("Successfully unregistered routes"))

			args = []string{"list", "--api", routingApiEndpoint, "--client-id", "tcp_emitter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).ShouldNot(Say(route))
		})
	})
})
