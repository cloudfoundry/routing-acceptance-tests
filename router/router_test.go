package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/assets/tcp-sample-receiver/testrunner"
	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/receptor"
)

const (
	DEFAULT_CONNECT_TIMEOUT = 1 * time.Second
	CONN_TYPE               = "tcp"
)

var _ = Describe("Routing Test", func() {
	var (
		externalPort        int
		sampleReceiverPort1 int
		sampleReceiverPort2 int
		serverId1           string
		serverId2           string

		receiver1 ifrit.Process
		receiver2 ifrit.Process
	)

	configureMapping := func(externalPort int, backendPorts ...int) {
		backends := cf_tcp_router.BackendHostInfos{}
		for _, backendPort := range backendPorts {
			backends = append(backends, cf_tcp_router.NewBackendHostInfo(externalIP, uint16(backendPort)))
		}

		createMappingRequest := cf_tcp_router.MappingRequests{
			cf_tcp_router.NewMappingRequest(uint16(externalPort), backends),
		}
		payload, err := json.Marshal(createMappingRequest)

		Expect(err).ToNot(HaveOccurred())

		resp, err := http.Post(fmt.Sprintf(
			"http://%s:%d/v0/external_ports",
			routerApiConfig.Address, routerApiConfig.Port),
			"application/json", bytes.NewBuffer(payload))
		Expect(err).ToNot(HaveOccurred())

		Expect(resp.StatusCode).Should(Equal(http.StatusOK))
	}

	checkConnection := func(errChan chan error, address string, serverId string) {
		time.Sleep(2 * time.Second)
		conn, err := net.DialTimeout(CONN_TYPE, address, DEFAULT_CONNECT_TIMEOUT)
		if err != nil {
			errChan <- err
			return
		}

		nanoSeconds := time.Now().Nanosecond()
		message := []byte(fmt.Sprintf("Time is %d", nanoSeconds))
		_, err = conn.Write(message)
		if err != nil {
			errChan <- err
			return
		}

		expectedMessage := []byte(serverId + ":" + string(message))
		buff := make([]byte, len(expectedMessage))
		_, err = conn.Read(buff)
		if err != nil {
			errChan <- err
			return
		}

		if !reflect.DeepEqual(buff, expectedMessage) {
			errChan <- errors.New(fmt.Sprintf("Message mismatch. Actual=[%s], Expected=[%s]", string(buff), string(expectedMessage)))
			return
		}
		errChan <- conn.Close()
	}

	verifyConnection := func(externalPort int, serverId string) {
		errChan := make(chan error, 1)
		address := fmt.Sprintf("%s:%d", routerApiConfig.Address, externalPort)
		go checkConnection(errChan, address, serverId)
		i := 0
	OUTERLOOP:
		for {
			select {
			case err := <-errChan:
				if err != nil {
					logger.Info(fmt.Sprintf("\n%d - Recevied error on errchan:%s\n", i, err.Error()))
					if i < 10 {
						i = i + 1
						go checkConnection(errChan, address, serverId)
					} else {
						Expect(err).ShouldNot(HaveOccurred())
					}
				} else {
					break OUTERLOOP
				}
			}
		}
	}

	spinupTcpReceiver := func(port int, id string) ifrit.Process {
		sampleReceiverArgs := testrunner.Args{
			Address:  fmt.Sprintf("%s:%d", externalIP, port),
			ServerId: id,
		}
		runner1 := testrunner.New(sampleReceiverPath, sampleReceiverArgs)
		return ifrit.Invoke(runner1)
	}

	tearDownTcpReceiver := func(receiverProcess ifrit.Process) {
		ginkgomon.Kill(receiverProcess, 5*time.Second)
	}

	Describe("A sample receiver running as a separate process", func() {
		BeforeEach(func() {
			externalPort = 60000 + GinkgoParallelNode()
			sampleReceiverPort1 = 9000 + GinkgoParallelNode()
			sampleReceiverPort2 = 9500 + GinkgoParallelNode()
			serverId1 = "serverId1"
			serverId2 = "serverId2"

			receiver1 = spinupTcpReceiver(sampleReceiverPort1, serverId1)
			receiver2 = spinupTcpReceiver(sampleReceiverPort2, serverId2)
		})

		AfterEach(func() {
			tearDownTcpReceiver(receiver1)
			tearDownTcpReceiver(receiver2)
		})

		It("routes traffic to sample receiver", func() {
			configureMapping(externalPort, sampleReceiverPort1)
			verifyConnection(externalPort, serverId1)

			By("altering the mapping it routes to new backend")
			configureMapping(externalPort, sampleReceiverPort2)
			verifyConnection(externalPort, serverId2)
		})
	})

	Describe("Multiple sample receivers running as a separate process and mapped to same external port", func() {

		sendAndReceive := func(address string) (net.Conn, string) {
			conn, err := net.DialTimeout(CONN_TYPE, address, DEFAULT_CONNECT_TIMEOUT)
			Expect(err).ShouldNot(HaveOccurred())

			message := "Hello"
			_, err = conn.Write([]byte(message))
			Expect(err).ShouldNot(HaveOccurred())

			response := make([]byte, 1024)
			count, err := conn.Read(response)
			Expect(err).ShouldNot(HaveOccurred())

			return conn, string(response[0:count])
		}

		BeforeEach(func() {
			externalPort = 61000 + GinkgoParallelNode()
			sampleReceiverPort1 = 7000 + GinkgoParallelNode()
			sampleReceiverPort2 = 7500 + GinkgoParallelNode()
			serverId1 = "serverId3"
			serverId2 = "serverId4"

			receiver1 = spinupTcpReceiver(sampleReceiverPort1, serverId1)
			receiver2 = spinupTcpReceiver(sampleReceiverPort2, serverId2)
		})

		AfterEach(func() {
			tearDownTcpReceiver(receiver1)
			tearDownTcpReceiver(receiver2)
		})

		It("load balances the connections", func() {
			configureMapping(externalPort, sampleReceiverPort1, sampleReceiverPort2)
			address := fmt.Sprintf("%s:%d", routerApiConfig.Address, externalPort)
			Eventually(func() error {
				tmpconn, err := net.Dial(CONN_TYPE, address)
				if err == nil {
					tmpconn.Close()
				}
				return err
			}, 20*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			conn1, response1 := sendAndReceive(address)
			conn2, response2 := sendAndReceive(address)
			Expect(response1).ShouldNot(Equal(response2))

			err := conn1.Close()
			Expect(err).ShouldNot(HaveOccurred())
			err = conn2.Close()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("LRP with TCP routing requirements is desired", func() {
		var (
			receptorClient receptor.Client
			processGuid    string
		)

		BeforeEach(func() {

			receptorClient = receptor.NewClient(routerApiConfig.DiegoAPIURL)

			externalPort = 62000 + GinkgoParallelNode()
			sampleReceiverPort1 = 8000 + GinkgoParallelNode()
			serverId1 = fmt.Sprintf("serverId-%d", GinkgoParallelNode())

			lrp := helpers.CreateDesiredLRP(logger, uint16(externalPort), uint16(sampleReceiverPort1), serverId1)

			err := receptorClient.CreateDesiredLRP(lrp)
			Expect(err).ShouldNot(HaveOccurred())
			processGuid = lrp.ProcessGuid
		})

		AfterEach(func() {
			err := receptorClient.DeleteDesiredLRP(processGuid)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("receives TCP traffic on desired external port", func() {
			verifyConnection(externalPort, serverId1)
		})
	})
})
