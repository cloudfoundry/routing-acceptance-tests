package router

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"

	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
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
)

var _ = SynchronizedBeforeSuite(func() []byte {
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
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
