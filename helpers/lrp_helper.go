package helpers

import (
	"fmt"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry-incubator/tcp-emitter/tcp_routes"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

func CreateDesiredLRP(logger lager.Logger, externalPort, containerPort uint16, serverId string) receptor.DesiredLRPCreateRequest {
	newProcessGuid, err := uuid.NewV4()
	if err != nil {
		logger.Error("failed-generate-guid", err)
		return receptor.DesiredLRPCreateRequest{}
	}
	route := tcp_routes.TCPRoute{
		ExternalPort:  externalPort,
		ContainerPort: containerPort,
	}
	routes := tcp_routes.TCPRoutes{route}
	lrp := receptor.DesiredLRPCreateRequest{
		ProcessGuid: newProcessGuid.String(),
		LogGuid:     "log-guid",
		Domain:      "tcp-routing-domain",
		Instances:   1,
		Setup: &models.SerialAction{
			Actions: []models.Action{
				&models.RunAction{
					Path: "sh",
					User: "vcap",
					Args: []string{
						"-c",
						"curl https://s3.amazonaws.com/router-release-blobs/tcp-sample-receiver.linux -o /tmp/tcp-sample-receiver && chmod +x /tmp/tcp-sample-receiver",
					},
				},
			},
		},
		Action: &models.ParallelAction{
			Actions: []models.Action{
				&models.RunAction{
					Path: "sh",
					User: "vcap",
					Args: []string{
						"-c",
						fmt.Sprintf("/tmp/tcp-sample-receiver -address 0.0.0.0:%d -serverId %s", containerPort, serverId),
					},
				},
			},
		},
		Monitor: &models.RunAction{
			Path: "sh",
			User: "vcap",
			Args: []string{
				"-c",
				fmt.Sprintf("nc -z 0.0.0.0 %d", containerPort),
			}},
		StartTimeout: 60,
		RootFS:       "preloaded:cflinuxfs2",
		MemoryMB:     128,
		DiskMB:       128,
		Ports:        []uint16{containerPort},
		Routes:       routes.RoutingInfo(),
		EgressRules: []models.SecurityGroupRule{
			{
				Protocol:     models.TCPProtocol,
				Destinations: []string{"0.0.0.0-255.255.255.255"},
				Ports:        []uint16{80, 443},
			},
			{
				Protocol:     models.UDPProtocol,
				Destinations: []string{"0.0.0.0/0"},
				PortRange: &models.PortRange{
					Start: 53,
					End:   53,
				},
			},
		},
	}
	return lrp
}
