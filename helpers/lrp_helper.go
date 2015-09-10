package helpers

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/tcp-emitter/tcp_routes"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

func CreateDesiredLRP(
	logger lager.Logger, externalPort,
	containerPort uint32, serverId string,
	instances int) *models.DesiredLRP {
	newProcessGuid, err := uuid.NewV4()
	if err != nil {
		logger.Error("failed-generate-guid", err)
		return nil
	}
	route := tcp_routes.TCPRoute{
		ExternalPort:  externalPort,
		ContainerPort: containerPort,
	}
	routes := tcp_routes.TCPRoutes{route}
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
					fmt.Sprintf("/tmp/tcp-sample-receiver -address 0.0.0.0:%d -serverId %s", containerPort, serverId),
				},
			},
		},
		Monitor: &models.Action{
			RunAction: &models.RunAction{
				Path: "sh",
				User: "vcap",
				Args: []string{
					"-c",
					fmt.Sprintf("nc -z 0.0.0.0 %d", containerPort),
				},
			},
		},
		StartTimeout: 60,
		RootFs:       "preloaded:cflinuxfs2",
		MemoryMb:     128,
		DiskMb:       128,
		Ports:        []uint32{containerPort},
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
	instances int) *models.DesiredLRPUpdate {
	route := tcp_routes.TCPRoute{
		ExternalPort:  externalPort,
		ContainerPort: containerPort,
	}
	routes := tcp_routes.TCPRoutes{route}
	numInstances := int32(instances)
	updatePayload := models.DesiredLRPUpdate{
		Instances: &numInstances,
		Routes:    routes.RoutingInfo(),
	}
	return &updatePayload
}
