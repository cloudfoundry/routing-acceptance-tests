package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

const (
	CREATE_ACTION         = "create"
	DELETE_ACTION         = "delete"
	SCALE_ACTION          = "scale"
	DEFAULT_DIEGO_API_URL = "http://10.244.16.46:8888"
)

var (
	logger lager.Logger
)

var diegoAPIURL = flag.String(
	"diegoAPIURL",
	DEFAULT_DIEGO_API_URL,
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
	1,
	"The desired number of instances.",
)

// ha proxy configuration
// listen cf-summit-proxy
//     mode tcp
//     bind :3456
//     server cellz10 10.244.16.10:61001

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

	receptorClient := receptor.NewClient(*diegoAPIURL)

	switch *action {
	case CREATE_ACTION:
		handleCreate(receptorClient)
	case DELETE_ACTION:
		handleDelete(receptorClient)
	case SCALE_ACTION:
		handleScale(receptorClient)
	default:
		logger.Fatal("unknown-parameter", errors.New(fmt.Sprintf("The command [%s] is not valid", *action)))
	}
}

func handleCreate(receptorClient receptor.Client) {
	newProcessGuid, err := uuid.NewV4()
	if err != nil {
		logger.Error("failed-generate-guid", err)
		return
	}
	route := tcpRoute{
		ExternalPort:  64000,
		ContainerPort: 5222,
	}
	routes := []tcpRoute{route}
	data, err := json.Marshal(routes)
	if err != nil {
		logger.Error("failed-to-marshal", err)
		return
	}
	routingInfo := json.RawMessage(data)
	lrp := receptor.DesiredLRPCreateRequest{
		ProcessGuid: newProcessGuid.String(),
		LogGuid:     "log-guid",
		Domain:      "ge",
		Instances:   1,
		Action: &models.ParallelAction{
			Actions: []models.Action{
				&models.RunAction{
					Path: "sh",
					Args: []string{
						"-c",
						"nc -l -k 5222 > /tmp/output",
					},
				},
			},
		},
		Monitor:      &models.RunAction{Path: "true"},
		StartTimeout: 60,
		RootFS:       "preloaded:cflinuxfs2",
		MemoryMB:     128,
		DiskMB:       128,
		Ports:        []uint16{5222},
		Routes: receptor.RoutingInfo{
			"tcp-router": &routingInfo,
		},
	}
	err = receptorClient.CreateDesiredLRP(lrp)
	if err != nil {
		logger.Error("failed-create", err, lager.Data{"LRP": lrp})
	} else {
		fmt.Printf("Successfully created LRP with process guid %s\n", newProcessGuid)
	}
}

func handleDelete(receptorClient receptor.Client) {
	if *processGuid == "" {
		logger.Fatal("missing-processGuid", errors.New("Missing mandatory processGuid parameter for delete action"))
	}

	err := receptorClient.DeleteDesiredLRP(*processGuid)
	if err != nil {
		logger.Error("failed-to-delete", err, lager.Data{"process-guid": *processGuid})
		return
	}
	fmt.Printf("Desired LRP successfully deleted for process guid %s\n", *processGuid)
}

func handleScale(receptorClient receptor.Client) {
	if *processGuid == "" {
		logger.Fatal("missing-processGuid", errors.New("Missing mandatory processGuid parameter for scale action"))
	}

	updatePayload := receptor.DesiredLRPUpdateRequest{
		Instances: numberOfInstances,
	}
	err := receptorClient.UpdateDesiredLRP(*processGuid, updatePayload)
	if err != nil {
		logger.Error("failed-to-scale", err, lager.Data{"process-guid": *processGuid, "update-request": updatePayload})
		return
	}
	fmt.Printf("LRP %s scaled to number of instances %d\n", *processGuid, *numberOfInstances)
}
