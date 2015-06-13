package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
)

const (
	DEFAULT_CONNECT_TIMEOUT = 1 * time.Second
	CONN_TYPE               = "tcp"
)

var _ = Describe("Routing Test", func() {

	Describe("A sample receiver running as a separate process", func() {
		var externalPort uint16

		BeforeEach(func() {
			externalPort = 52000
			backends := cf_tcp_router.BackendHostInfos{
				cf_tcp_router.NewBackendHostInfo(externalIP, uint16(sampleReceiverPort)),
			}
			createMappingRequest := cf_tcp_router.MappingRequests{
				cf_tcp_router.NewMappingRequest(externalPort, backends),
			}
			payload, err := json.Marshal(createMappingRequest)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(fmt.Sprintf(
				"http://%s:%d/v0/external_ports",
				routerApiConfig.Address, routerApiConfig.Port),
				"application/json", bytes.NewBuffer(payload))
			Expect(err).ToNot(HaveOccurred())

			Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		})

		It("Routes traffic to sample receiver", func() {
			address := fmt.Sprintf("%s:%d", routerApiConfig.Address, externalPort)
			Eventually(func() error {
				tmpconn, err := net.Dial(CONN_TYPE, address)
				if err == nil {
					tmpconn.Close()
				}
				return err
			}, 10*time.Second).ShouldNot(HaveOccurred())

			conn, err := net.DialTimeout(CONN_TYPE, address, DEFAULT_CONNECT_TIMEOUT)
			Expect(err).ShouldNot(HaveOccurred())

			nanoSeconds := time.Now().Nanosecond()

			message := []byte(fmt.Sprintf("Time is %s", nanoSeconds))
			_, err = conn.Write(message)
			Expect(err).ShouldNot(HaveOccurred())

			buff := make([]byte, len(message))
			_, err = conn.Read(buff)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(buff).Should(Equal(message))
			conn.Close()
		})
	})
})
