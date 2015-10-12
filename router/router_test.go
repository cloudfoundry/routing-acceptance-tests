package router

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"github.com/cloudfoundry-incubator/bbs"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/assets/tcp-sample-receiver/testrunner"
	"github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/tcp-emitter/tcp_routes"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

const (
	DEFAULT_CONNECT_TIMEOUT = 1 * time.Second
	CONN_TYPE               = "tcp"
)

var _ = Describe("Routing Test", func() {

	var (
		externalPort1       int
		externalPort2       int
		sampleReceiverPort1 int
		sampleReceiverPort2 int
		serverId1           string
		serverId2           string

		receiver1        ifrit.Process
		receiver2        ifrit.Process
		routingApiClient routing_api.Client
	)

	const (
		ROUTER_GROUP_1 = "rtr-grp-1"
	)

	BeforeEach(func() {
		routingApiClient = routing_api.NewClient(routerApiConfig.RoutingApiUrl)
		oauth := token_fetcher.OAuthConfig{
			TokenEndpoint: routerApiConfig.OAuth.TokenEndpoint,
			ClientName:    routerApiConfig.OAuth.ClientName,
			ClientSecret:  routerApiConfig.OAuth.ClientSecret,
			Port:          routerApiConfig.OAuth.Port,
		}
		tokenFetcher := token_fetcher.NewTokenFetcher(&oauth)
		token, err := tokenFetcher.FetchToken()
		Expect(err).ToNot(HaveOccurred())
		routingApiClient.SetToken(token.AccessToken)
	})

	getTcpRouteMappings := func(externalPort int, backendPorts ...int) []db.TcpRouteMapping {
		tcpRouteMappings := make([]db.TcpRouteMapping, 0)

		for _, backendPort := range backendPorts {
			tcpMapping := db.NewTcpRouteMapping(ROUTER_GROUP_1, uint16(externalPort), externalIP, uint16(backendPort))
			tcpRouteMappings = append(tcpRouteMappings, tcpMapping)
		}
		return tcpRouteMappings
	}

	configureRoutingApiMapping := func(externalPort int, backendPorts ...int) {
		tcpRouteMappings := getTcpRouteMappings(externalPort, backendPorts...)

		err := routingApiClient.UpsertTcpRouteMappings(tcpRouteMappings)
		Expect(err).ToNot(HaveOccurred())
	}

	deleteRoutingApiMapping := func(externalPort int, backendPorts ...int) {
		tcpRouteMappings := getTcpRouteMappings(externalPort, backendPorts...)

		err := routingApiClient.DeleteTcpRouteMappings(tcpRouteMappings)
		Expect(err).ToNot(HaveOccurred())
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
		Context("using routing api", func() {
			BeforeEach(func() {
				externalPort1 = 60500 + GinkgoParallelNode()
				sampleReceiverPort1 = 10500 + GinkgoParallelNode()
				sampleReceiverPort2 = 11000 + GinkgoParallelNode()
				serverId1 = "serverId-1-routing-api"
				serverId2 = "serverId-2-routing-api"

				receiver1 = spinupTcpReceiver(sampleReceiverPort1, serverId1)
				receiver2 = spinupTcpReceiver(sampleReceiverPort2, serverId2)
			})

			AfterEach(func() {
				tearDownTcpReceiver(receiver1)
				tearDownTcpReceiver(receiver2)

				deleteRoutingApiMapping(externalPort1, sampleReceiverPort1)
				deleteRoutingApiMapping(externalPort1, sampleReceiverPort2)
			})

			It("routes traffic to sample receiver", func() {
				configureRoutingApiMapping(externalPort1, sampleReceiverPort1)
				verifyConnection(externalPort1, serverId1)

				By("altering the mapping it routes to new backend")
				configureRoutingApiMapping(externalPort1, sampleReceiverPort2)
				verifyConnection(externalPort1, serverId2)
			})
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

		Context("using routing api", func() {
			BeforeEach(func() {
				externalPort1 = 61500 + GinkgoParallelNode()
				sampleReceiverPort1 = 11000 + GinkgoParallelNode()
				sampleReceiverPort2 = 11500 + GinkgoParallelNode()
				serverId1 = "serverId-1-multiple-receivers-routing-api"
				serverId2 = "serverId-2-multiple-receivers-routing-api"

				receiver1 = spinupTcpReceiver(sampleReceiverPort1, serverId1)
				receiver2 = spinupTcpReceiver(sampleReceiverPort2, serverId2)
			})

			AfterEach(func() {
				tearDownTcpReceiver(receiver1)
				tearDownTcpReceiver(receiver2)

				deleteRoutingApiMapping(externalPort1, sampleReceiverPort1, sampleReceiverPort2)
			})

			It("load balances the connections", func() {
				configureRoutingApiMapping(externalPort1, sampleReceiverPort1, sampleReceiverPort2)

				address := fmt.Sprintf("%s:%d", routerApiConfig.Address, externalPort1)

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

	})

	Describe("LRP mapped to multiple external ports", func() {
		var (
			bbsClient   bbs.Client
			processGuid string
		)

		createDesiredLRPTwoExternalPorts := func(
			externalPort1,
			externalPort2,
			sampleReceiverPort1 uint32,
			serverId string) *models.DesiredLRP {
			lrp := helpers.CreateDesiredLRP(logger,
				externalPort1, sampleReceiverPort1, serverId1, 1)

			route1 := tcp_routes.TCPRoute{
				ExternalPort:  externalPort1,
				ContainerPort: sampleReceiverPort1,
			}
			route2 := tcp_routes.TCPRoute{
				ExternalPort:  externalPort2,
				ContainerPort: sampleReceiverPort1,
			}
			routes := tcp_routes.TCPRoutes{route1, route2}
			lrp.Routes = routes.RoutingInfo()
			return lrp
		}

		BeforeEach(func() {
			var bbsErr error
			bbsClient, bbsErr = helpers.GetBbsClient(routerApiConfig)
			Expect(bbsErr).ToNot(HaveOccurred())

			externalPort1 = 34500 + GinkgoParallelNode()
			externalPort2 = 12300 + GinkgoParallelNode()

			sampleReceiverPort1 = 7000 + GinkgoParallelNode()
			serverId1 = "serverId6"

			lrp := createDesiredLRPTwoExternalPorts(
				uint32(externalPort1),
				uint32(externalPort2),
				uint32(sampleReceiverPort1),
				serverId1,
			)
			err := bbsClient.DesireLRP(lrp)
			Expect(err).ShouldNot(HaveOccurred())
			processGuid = lrp.ProcessGuid
		})

		AfterEach(func() {
			err := bbsClient.RemoveDesiredLRP(processGuid)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("sends traffic on the different external ports to the same container port", func() {
			verifyConnection(externalPort1, serverId1)
			verifyConnection(externalPort2, serverId1)
		})
	})

	Describe("LRP with TCP routing requirements is desired", func() {
		var (
			bbsClient   bbs.Client
			processGuid string
		)

		BeforeEach(func() {
			var bbsErr error
			bbsClient, bbsErr = helpers.GetBbsClient(routerApiConfig)
			Expect(bbsErr).ToNot(HaveOccurred())

			externalPort1 = 62000 + GinkgoParallelNode()
			sampleReceiverPort1 = 8000 + GinkgoParallelNode()
			serverId1 = fmt.Sprintf("serverId-%d", GinkgoParallelNode())

			lrp := helpers.CreateDesiredLRP(logger,
				uint32(externalPort1), uint32(sampleReceiverPort1), serverId1, 1)

			err := bbsClient.DesireLRP(lrp)
			Expect(err).ShouldNot(HaveOccurred())
			processGuid = lrp.ProcessGuid
		})

		AfterEach(func() {
			err := bbsClient.RemoveDesiredLRP(processGuid)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("receives TCP traffic on desired external port", func() {
			verifyConnection(externalPort1, serverId1)

			By("updating LRP with new external port it receives traffic on new external port")
			externalPort1 = 63000 + GinkgoParallelNode()
			updatedLrp := helpers.UpdateDesiredLRP(uint32(externalPort1),
				uint32(sampleReceiverPort1), 1)
			err := bbsClient.UpdateDesiredLRP(processGuid, updatedLrp)
			Expect(err).ShouldNot(HaveOccurred())
			verifyConnection(externalPort1, serverId1)
		})
	})
})
