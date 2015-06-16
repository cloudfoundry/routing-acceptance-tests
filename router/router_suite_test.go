package router

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/assets/tcp-sample-receiver/testrunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"

	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
)

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Suite")
}

var (
	sampleReceiverPort    int
	sampleReceiverPath    string
	externalIP            string
	sampleReceiverProcess ifrit.Process
	routerApiConfig       helpers.RouterApiConfig
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

	sampleReceiverPort = 7000 + GinkgoParallelNode()
	sampleReceiverPath = context["sample-receiver"]
	externalIP = testutil.GetExternalIP()
	routerApiConfig = helpers.LoadConfig()
})

var _ = BeforeEach(func() {
	sampleReceiverArgs := testrunner.Args{
		Address: fmt.Sprintf("%s:%d", externalIP, sampleReceiverPort),
	}

	runner := testrunner.New(sampleReceiverPath, sampleReceiverArgs)
	sampleReceiverProcess = ifrit.Invoke(runner)
})

var _ = AfterEach(func() {
	ginkgomon.Kill(sampleReceiverProcess, 5*time.Second)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
