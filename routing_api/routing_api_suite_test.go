package routing_api

import (
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-router-acceptance-tests/helpers"

	"testing"
)

func Rtr(args ...string) *Session {
	session, err := Start(exec.Command("rtr", args...), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}

const (
	DEFAULT_TIMEOUT      = 30 * time.Second
	CF_PUSH_TIMEOUT      = 2 * time.Minute
	DEFAULT_MEMORY_LIMIT = "256M"
)

var routerApiConfig helpers.RouterApiConfig

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	routerApiConfig = helpers.LoadConfig()

	BeforeSuite(func() {
		Expect(routerApiConfig.SystemDomain).ToNot(Equal(""), "Must provide a system domain for the routing suite")
		Expect(routerApiConfig.OAuth.ClientSecret).ToNot(Equal(""), "Must provide a client secret for the routing suite")
	})

	RunSpecs(t, "Routing API Suite")
}
