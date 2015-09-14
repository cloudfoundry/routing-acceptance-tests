package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/cloudfoundry-incubator/bbs"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

const (
	CREATE_ACTION          = "create"
	UPDATE_ACTION          = "update"
	DELETE_ACTION          = "delete"
	DEFAULT_BBS_ADDRESS    = "http://10.244.16.130:8889"
	DEFAULT_SERVER_ID      = "server-1"
	DEFAULT_EXTERNAL_PORT  = 64000
	DEFAULT_CONTAINER_PORT = 5222
)

var (
	logger lager.Logger
)

var serverId = flag.String(
	"serverId",
	DEFAULT_SERVER_ID,
	"ID Of the server being created via Diego",
)

var externalPort = flag.Int(
	"externalPort",
	0,
	"The external port.",
)

var containerPort = flag.Int(
	"containerPort",
	DEFAULT_CONTAINER_PORT,
	"The container port.",
)

var bbsAddress = flag.String(
	"bbsAddress",
	DEFAULT_BBS_ADDRESS,
	"URL of diego API",
)

var action = flag.String(
	"action",
	"",
	"The action can be: create, delete or scale.",
)

var processGuid = flag.String(
	"processGuid",
	"",
	"The process GUID of the target LRP.",
)

var numberOfInstances = flag.Int(
	"instances",
	-1,
	"The desired number of instances.",
)

type tcpRoute struct {
	ExternalPort  uint16 `json:"external_port"`
	ContainerPort uint16 `json:"container_port"`
}

func main() {
	cf_lager.AddFlags(flag.CommandLine)
	logger, _ = cf_lager.New("desiredlrp-client")

	flag.Parse()

	if *action == "" {
		logger.Fatal("action-required", errors.New("Missing mandatory action parameter"))
	}

	bbsClient := bbs.NewClient(*bbsAddress)

	switch *action {
	case CREATE_ACTION:
		handleCreate(bbsClient)
	case DELETE_ACTION:
		handleDelete(bbsClient)
	case UPDATE_ACTION:
		handleUpdate(bbsClient)
	default:
		logger.Fatal("unknown-parameter", errors.New(fmt.Sprintf("The command [%s] is not valid", *action)))
	}
}

func handleCreate(bbsClient bbs.Client) {
	newProcessGuid, err := uuid.NewV4()
	if err != nil {
		logger.Error("failed-generate-guid", err)
		return
	}
	extPort := *externalPort
	if extPort == 0 {
		extPort = DEFAULT_EXTERNAL_PORT
	}
	route := tcpRoute{
		ExternalPort:  uint16(extPort),
		ContainerPort: uint16(*containerPort),
	}
	routes := []tcpRoute{route}
	data, err := json.Marshal(routes)
	if err != nil {
		logger.Error("failed-to-marshal", err)
		return
	}
	routingInfo := json.RawMessage(data)
	lrp := models.DesiredLRP{
		ProcessGuid: newProcessGuid.String(),
		LogGuid:     "log-guid",
		Domain:      "ge",
		Instances:   1,
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
					fmt.Sprintf("/tmp/tcp-sample-receiver -address 0.0.0.0:%d -serverId %s", *containerPort, *serverId),
					// fmt.Sprintf("nc -l -k %d > /tmp/output", *containerPort),
				},
			},
		},
		Monitor: &models.Action{
			RunAction: &models.RunAction{
				Path: "sh",
				User: "vcap",
				Args: []string{
					"-c",
					fmt.Sprintf("nc -z 0.0.0.0 %d", *containerPort),
				},
			},
		},
		StartTimeout: 60,
		RootFs:       "preloaded:cflinuxfs2",
		MemoryMb:     128,
		DiskMb:       128,
		Ports:        []uint32{uint32(*containerPort)},
		Routes: &models.Routes{
			"tcp-router": &routingInfo,
		},
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
	err = bbsClient.DesireLRP(&lrp)
	if err != nil {
		logger.Error("failed-create", err, lager.Data{"LRP": lrp})
	} else {
		fmt.Printf("Successfully created LRP with process guid %s\n", newProcessGuid)
	}
}

func handleDelete(bbsClient bbs.Client) {
	if *processGuid == "" {
		logger.Fatal("missing-processGuid", errors.New("Missing mandatory processGuid parameter for delete action"))
	}

	err := bbsClient.RemoveDesiredLRP(*processGuid)
	if err != nil {
		logger.Error("failed-to-delete", err, lager.Data{"process-guid": *processGuid})
		return
	}
	fmt.Printf("Desired LRP successfully deleted for process guid %s\n", *processGuid)
}

func handleUpdate(bbsClient bbs.Client) {
	if *processGuid == "" {
		logger.Fatal("missing-processGuid", errors.New("Missing mandatory processGuid parameter for scale action"))
	}

	updated := false
	var updatePayload models.DesiredLRPUpdate
	if *numberOfInstances >= 0 {
		instances := int32(*numberOfInstances)
		updatePayload.Instances = &instances
		updated = true
	}

	if *externalPort > 0 {
		route := tcpRoute{
			ExternalPort:  uint16(*externalPort),
			ContainerPort: uint16(*containerPort),
		}
		routes := []tcpRoute{route}
		data, err := json.Marshal(routes)
		if err != nil {
			logger.Error("failed-to-marshal", err)
			return
		}
		routingInfo := json.RawMessage(data)
		updatePayload.Routes = &models.Routes{
			"tcp-router": &routingInfo,
		}
		updated = true
	}

	if updated {
		err := bbsClient.UpdateDesiredLRP(*processGuid, &updatePayload)
		if err != nil {
			logger.Error("failed-to-scale", err, lager.Data{"process-guid": *processGuid, "update-request": updatePayload})
			return
		}
		fmt.Printf("LRP %s updated \n", *processGuid)
	}
}
