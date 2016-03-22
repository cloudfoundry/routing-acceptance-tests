package routing_api

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Registration", func() {
	var (
		route     string
		routeJSON string
	)

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
			args := []string{"events", "--http"}
			eventsSession = Rtr(args...)
			Eventually(eventsSession.Out, 70*time.Second).Should(Say("port"))
			Eventually(eventsSession.Out, 70*time.Second).Should(Say("route"))

			args = []string{"register", routeJSON}
			session := Rtr(args...)
			Eventually(session.Out).Should(Say("Successfully registered routes"))
			Eventually(eventsSession.Out, 10*time.Second).Should(Say(route))
			Eventually(eventsSession.Out).Should(Say(`"port":65340`))
			Eventually(eventsSession.Out).Should(Say(`"ip":"1.2.3.4"`))
			Eventually(eventsSession.Out).Should(Say(`"ttl":60`))
			Eventually(eventsSession.Out).Should(Say(`"Action":"Upsert"`))

			args = []string{"list"}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say(route))

			args = []string{"unregister", routeJSON}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say("Successfully unregistered routes"))

			args = []string{"list"}
			session = Rtr(args...)

			Eventually(session.Out).ShouldNot(Say(route))
		})
	})
})
