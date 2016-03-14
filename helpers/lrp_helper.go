package helpers

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/routing-info/tcp_routes"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

func CreateDesiredLRP(
	logger lager.Logger, containerPorts []uint32,
	routes tcp_routes.TCPRoutes, serverId string,
	instances int) *models.DesiredLRP {
	newProcessGuid, err := uuid.NewV4()
	if err != nil {
		logger.Error("failed-generate-guid", err)
		return nil
	}
	addresses := make([]string, 0)
	for _, containerPort := range containerPorts {
		addresses = append(addresses, fmt.Sprintf("0.0.0.0:%d", containerPort))
	}
	address := strings.Join(addresses, ",")
	lrp := models.DesiredLRP{
		ProcessGuid: newProcessGuid.String(),
		LogGuid:     "log-guid",
		Domain:      "tcp-routing-domain",
		Instances:   int32(instances),
		Setup: &models.Action{
			RunAction: &models.RunAction{
				Path: "sh",
				User: "vcap",
				Args: []string{
					"-c",
					"curl https://s3.amazonaws.com/router-release-blobs/tcp-sample-receiver.linux -o /tmp/tcp-sample-receiver && chmod +x /tmp/tcp-sample-receiver",
				},
			},
		},
		Action: &models.Action{
			RunAction: &models.RunAction{
				Path: "sh",
				User: "vcap",
				Args: []string{
					"-c",
					fmt.Sprintf("/tmp/tcp-sample-receiver -address %s -serverId %s", address, serverId),
				},
			},
		},
		Monitor: &models.Action{
			RunAction: &models.RunAction{
				Path: "sh",
				User: "vcap",
				Args: []string{
					"-c",
					fmt.Sprintf("nc -z 0.0.0.0 %d", containerPorts[0]),
				},
			},
		},
		StartTimeout: 60,
		RootFs:       "preloaded:cflinuxfs2",
		MemoryMb:     128,
		DiskMb:       128,
		Ports:        containerPorts,
		Routes:       routes.RoutingInfo(),
		EgressRules: []*models.SecurityGroupRule{
			&models.SecurityGroupRule{
				Protocol:     "tcp",
				Destinations: []string{"0.0.0.0-255.255.255.255"},
				Ports:        []uint32{80, 443},
			},
			&models.SecurityGroupRule{
				Protocol:     "udp",
				Destinations: []string{"0.0.0.0/0"},
				PortRange: &models.PortRange{
					Start: 53,
					End:   53,
				},
			},
		},
	}
	return &lrp
}

func UpdateDesiredLRP(
	externalPort, containerPort uint32,
	instances int, routerGroupGuid string) *models.DesiredLRPUpdate {
	route := tcp_routes.TCPRoute{
		RouterGroupGuid: routerGroupGuid,
		ExternalPort:    externalPort,
		ContainerPort:   containerPort,
	}
	routes := tcp_routes.TCPRoutes{route}
	numInstances := int32(instances)
	updatePayload := models.DesiredLRPUpdate{
		Instances: &numInstances,
		Routes:    routes.RoutingInfo(),
	}
	return &updatePayload
}
