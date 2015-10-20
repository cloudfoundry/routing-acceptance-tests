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
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/tcp-emitter/tcp_routes"
)

const (
	DEFAULT_CONNECT_TIMEOUT = 1 * time.Second
	CONN_TYPE               = "tcp"
)

var _ = Describe("Routing Test", func() {

	var (
		serverId1 string
		serverId2 string

		receiver1 ifrit.Process
		receiver2 ifrit.Process
	)

	const (
		ROUTER_GROUP_1 = "rtr-grp-1"
	)

	isLRPRunning := func(bbsClient bbs.Client, processGuid string) bool {
		actualLrps, err := bbsClient.ActualLRPGroupsByProcessGuid(processGuid)
		if err != nil {
			return false
		}
		return len(actualLrps) > 0 &&
			actualLrps[0].Instance.State == models.ActualLRPStateRunning
	}

	isLRPRemoved := func(bbsClient bbs.Client, processGuid string) bool {
		actualLrps, err := bbsClient.ActualLRPGroupsByProcessGuid(processGuid)
		if err != nil {
			return false
		}
		return len(actualLrps) == 0
	}

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

	verifyConnection := func(externalPort int, serverId string, addr string) {
		errChan := make(chan error, 1)
		address := fmt.Sprintf("%s:%d", addr, externalPort)
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

	verifyPortClosedOnAddress := func(externalPort int, addr string) bool {
		address := fmt.Sprintf("%s:%d", addr, externalPort)
		conn, _ := net.DialTimeout(CONN_TYPE, address, DEFAULT_CONNECT_TIMEOUT)
		defer func() {
			if conn != nil {
				conn.Close()
			}
		}()
		return conn == nil
	}

	verifyPortClosed := func(port int) {
		for _, address := range routerApiConfig.Addresses {
			Eventually(func() bool {
				return verifyPortClosedOnAddress(port, address)
			}, 30*time.Second, 5*time.Second).Should(BeTrue(),
				fmt.Sprintf("port %d not closed", port))
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

	verifyConnections := func(externalPort int, serverId string) {
		for _, address := range routerApiConfig.Addresses {
			verifyConnection(externalPort, serverId, address)
		}
	}

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

	verifyLoadBalancing := func(externalPort int, addr string, expectedResponses []string) {
		address := fmt.Sprintf("%s:%d", addr, externalPort)

		Eventually(func() error {
			tmpconn, err := net.Dial(CONN_TYPE, address)
			if err == nil {
				tmpconn.Close()
			}
			return err
		}, 40*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		conn1, response1 := sendAndReceive(address)
		conn2, response2 := sendAndReceive(address)
		Expect(response1).ShouldNot(Equal(response2))

		Expect(len(expectedResponses)).Should(Equal(2))
		Expect(expectedResponses).Should(ContainElement(response1))
		Expect(expectedResponses).Should(ContainElement(response2))

		err := conn1.Close()
		Expect(err).ShouldNot(HaveOccurred())
		err = conn2.Close()
		Expect(err).ShouldNot(HaveOccurred())
	}

	Describe("A sample receiver running as a separate process", func() {
		var (
			externalPort1       int
			sampleReceiverPort1 int
			sampleReceiverPort2 int
		)

		Context("using routing api", func() {
			BeforeEach(func() {
				externalPort1 = nextExternalPort() + GinkgoParallelNode()
				sampleReceiverPort1 = nextContainerPort() + GinkgoParallelNode()
				sampleReceiverPort2 = nextContainerPort() + GinkgoParallelNode()
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
				verifyPortClosed(externalPort1)
			})

			It("routes traffic to sample receiver", func() {
				configureRoutingApiMapping(externalPort1, sampleReceiverPort1)
				verifyConnections(externalPort1, serverId1)

				By("altering the mapping it routes to new backend")
				configureRoutingApiMapping(externalPort1, sampleReceiverPort2)
				verifyConnections(externalPort1, serverId2)
			})
		})
	})

	Describe("Multiple sample receivers running as a separate process and mapped to same external port", func() {
		var (
			externalPort1       int
			sampleReceiverPort1 int
			sampleReceiverPort2 int
		)

		Context("using routing api", func() {
			BeforeEach(func() {
				externalPort1 = nextExternalPort() + GinkgoParallelNode()
				sampleReceiverPort1 = nextContainerPort() + GinkgoParallelNode()
				sampleReceiverPort2 = nextContainerPort() + GinkgoParallelNode()
				serverId1 = "serverId-1-multiple-receivers-routing-api"
				serverId2 = "serverId-2-multiple-receivers-routing-api"

				receiver1 = spinupTcpReceiver(sampleReceiverPort1, serverId1)
				receiver2 = spinupTcpReceiver(sampleReceiverPort2, serverId2)
			})

			AfterEach(func() {
				tearDownTcpReceiver(receiver1)
				tearDownTcpReceiver(receiver2)

				deleteRoutingApiMapping(externalPort1, sampleReceiverPort1, sampleReceiverPort2)
				verifyPortClosed(externalPort1)
			})

			It("load balances the connections", func() {
				configureRoutingApiMapping(externalPort1, sampleReceiverPort1, sampleReceiverPort2)
				expectedResponses := []string{fmt.Sprintf("%s:Hello", serverId1), fmt.Sprintf("%s:Hello", serverId2)}
				for _, address := range routerApiConfig.Addresses {
					verifyLoadBalancing(externalPort1, address, expectedResponses)
				}
			})
		})

	})

	Describe("LRP", func() {
		Context("mapped to multiple external ports", func() {
			var (
				bbsClient           bbs.Client
				processGuid         string
				externalPort1       int
				externalPort2       int
				sampleReceiverPort1 int
			)

			createDesiredLRPTwoExternalPorts := func(
				externalPort1,
				externalPort2,
				sampleReceiverPort1 uint32,
				serverId string) *models.DesiredLRP {
				containerPorts := []uint32{sampleReceiverPort1}
				route1 := tcp_routes.TCPRoute{
					RouterGroupGuid: "bad25cff-9332-48a6-8603-b619858e7992",
					ExternalPort:    externalPort1,
					ContainerPort:   sampleReceiverPort1,
				}
				route2 := tcp_routes.TCPRoute{
					RouterGroupGuid: "bad25cff-9332-48a6-8603-b619858e7992",
					ExternalPort:    externalPort2,
					ContainerPort:   sampleReceiverPort1,
				}
				routes := tcp_routes.TCPRoutes{route1, route2}
				lrp := helpers.CreateDesiredLRP(logger, containerPorts, routes, serverId1, 1)
				return lrp
			}

			BeforeEach(func() {
				var bbsErr error
				bbsClient, bbsErr = helpers.GetBbsClient(routerApiConfig)
				Expect(bbsErr).ToNot(HaveOccurred())

				externalPort1 = nextExternalPort() + GinkgoParallelNode()
				externalPort2 = nextExternalPort() + GinkgoParallelNode()
				sampleReceiverPort1 = nextContainerPort() + GinkgoParallelNode()

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
				Eventually(func() bool {
					return isLRPRunning(bbsClient, processGuid)
				}, 30*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("LRP (%s) not running", processGuid))
			})

			AfterEach(func() {
				err := bbsClient.RemoveDesiredLRP(processGuid)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() bool {
					return isLRPRemoved(bbsClient, processGuid)
				}, 30*time.Second, 1*time.Second).Should(BeTrue())
				verifyPortClosed(externalPort1)
				verifyPortClosed(externalPort2)

			})

			It("sends traffic on the different external ports to the same container port", func() {
				verifyConnections(externalPort1, serverId1)
				verifyConnections(externalPort2, serverId1)
			})
		})

		Context("mapped to single external port", func() {
			var (
				bbsClient     bbs.Client
				processGuid   string
				lrp           *models.DesiredLRP
				externalPort1 int
			)
			BeforeEach(func() {
				var bbsErr error
				bbsClient, bbsErr = helpers.GetBbsClient(routerApiConfig)
				Expect(bbsErr).ToNot(HaveOccurred())
				externalPort1 = nextExternalPort() + GinkgoParallelNode()
			})

			JustBeforeEach(func() {
				err := bbsClient.DesireLRP(lrp)
				Expect(err).ShouldNot(HaveOccurred())
				processGuid = lrp.ProcessGuid
				Eventually(func() bool {
					return isLRPRunning(bbsClient, processGuid)
				}, 30*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("LRP (%s) not running", processGuid))
			})

			AfterEach(func() {
				err := bbsClient.RemoveDesiredLRP(processGuid)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() bool {
					return isLRPRemoved(bbsClient, processGuid)
				}, 30*time.Second, 1*time.Second).Should(BeTrue())
				verifyPortClosed(externalPort1)
			})

			Context("with one container port", func() {
				var (
					sampleReceiverPort1 int
				)
				BeforeEach(func() {
					sampleReceiverPort1 = nextContainerPort() + GinkgoParallelNode()
					serverId1 = fmt.Sprintf("serverId-%d", GinkgoParallelNode())

					containerPorts := []uint32{uint32(sampleReceiverPort1)}
					route1 := tcp_routes.TCPRoute{
						RouterGroupGuid: "bad25cff-9332-48a6-8603-b619858e7992",
						ExternalPort:    uint32(externalPort1),
						ContainerPort:   uint32(sampleReceiverPort1),
					}

					routes := tcp_routes.TCPRoutes{route1}
					lrp = helpers.CreateDesiredLRP(logger, containerPorts, routes, serverId1, 1)
				})

				It("receives TCP traffic on desired external port", func() {
					oldExternalPort := externalPort1
					verifyConnections(externalPort1, serverId1)

					By("updating LRP with new external port it receives traffic on new external port")
					externalPort1 = nextExternalPort() + GinkgoParallelNode()
					updatedLrp := helpers.UpdateDesiredLRP(uint32(externalPort1),
						uint32(sampleReceiverPort1), 1)
					err := bbsClient.UpdateDesiredLRP(processGuid, updatedLrp)
					Expect(err).ShouldNot(HaveOccurred())
					verifyConnections(externalPort1, serverId1)

					verifyPortClosed(oldExternalPort)
				})
			})

			Context("with multiple container ports", func() {
				var (
					externalPort1       int
					sampleReceiverPort1 int
					sampleReceiverPort2 int
				)
				BeforeEach(func() {
					externalPort1 = nextExternalPort() + GinkgoParallelNode()
					sampleReceiverPort1 = nextContainerPort() + GinkgoParallelNode()
					sampleReceiverPort2 = nextContainerPort() + GinkgoParallelNode()
					serverId1 = fmt.Sprintf("serverId-%d", GinkgoParallelNode())

					containerPorts := []uint32{uint32(sampleReceiverPort1), uint32(sampleReceiverPort2)}
					route1 := tcp_routes.TCPRoute{
						RouterGroupGuid: "bad25cff-9332-48a6-8603-b619858e7992",
						ExternalPort:    uint32(externalPort1),
						ContainerPort:   uint32(sampleReceiverPort1),
					}

					routes := tcp_routes.TCPRoutes{route1}
					lrp = helpers.CreateDesiredLRP(logger, containerPorts, routes, serverId1, 1)
				})

				It("receives TCP traffic on desired external port", func() {
					prefix := serverId1 + fmt.Sprintf("(0.0.0.0:%d)", sampleReceiverPort1)
					verifyConnections(externalPort1, prefix)

					By("updating LRP to map external port to different container port")
					updatedLrp := helpers.UpdateDesiredLRP(uint32(externalPort1),
						uint32(sampleReceiverPort2), 1)
					err := bbsClient.UpdateDesiredLRP(processGuid, updatedLrp)
					Expect(err).ShouldNot(HaveOccurred())

					prefix = serverId1 + fmt.Sprintf("(0.0.0.0:%d)", sampleReceiverPort2)
					verifyConnections(externalPort1, prefix)
				})
			})
		})
	})
})
