package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/health"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/versions"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	"k8s.io/klog"
)

type MocResourceType string

const (
	LoadBalancer     MocResourceType = "LoadBalancer"
	VipPool          MocResourceType = "VipPool"
	VirtualNetwork   MocResourceType = "VirtualNetwork"
	NetworkInterface MocResourceType = "NetworkInterface"
	Disk             MocResourceType = "Disk"
	VirtualMachine   MocResourceType = "VirtualMachine"
	KeyVault         MocResourceType = "KeyVault"
	Secret           MocResourceType = "Secret"
	Group            MocResourceType = "Group"
)

type MocOperation string

const (
	CreateOrUpdate MocOperation = "CreateOrUpdate"
	Delete         MocOperation = "Delete"
	Get            MocOperation = "Get"
)

type OperationLog struct {
	Timestamp      string      `json:"timestamp"`
	ParentResource string      `json:"parent_resource"`
	Resource       string      `json:"resource"`
	FilterKeyword  string      `json:"filter_keyword"`
	Action         string      `json:"action"`
	Params         interface{} `json:"params"`
	ErrorCode      string      `json:"error_code"`
	Message        string      `json:"msg"`
}

func WriteMocOperationLog(operation MocOperation, crResourceName string, mocResourceType MocResourceType, mocResourceName string, params interface{}, err error) {
	errcode := "0"
	message := ""
	if err != nil {
		errcode = mocerrors.GetErrorCode(err)
		message = err.Error()
	}

	oplog := OperationLog{
		Timestamp:      time.Now().Format(time.RFC3339),
		ParentResource: crResourceName,
		Resource:       fmt.Sprintf("%s/%s", mocResourceType, mocResourceName),
		FilterKeyword:  "RESOURCE_ACTION",
		Action:         string(operation),
		Params:         params,
		ErrorCode:      errcode,
		Message:        message,
	}

	jsonData, err := json.Marshal(oplog)
	if err != nil {
		klog.Error("Unable to serialize operation log object. ", crResourceName)
	} else {
		klog.Info(string(jsonData))
	}
}

func GenerateMocResourceName(nameSegments ...string) string {
	return strings.Join(nameSegments, "/")
}

type MocInfoLog struct {
	MocDeploymentId       string `json:"moc_deployment_id"`
	WssdCloudAgentVersion string `json:"wssd_cloud_agent_version"`
	MocVersion            string `json:"moc_version"`
}

var healthService *health.Service
var versionsService *versions.Service

func WriteMocInfoLog(ctx context.Context, scope scope.ScopeInterface) {
	deploymentId := getHealthService(scope).GetMocDeploymentId(ctx)
	wssdCloudAgentVersion := ""
	mocVersion := ""

	versionPair, err := getVersionsService(scope).Get(ctx)
	if err != nil {
		klog.Error("Unable to get moc version. ", err)
	} else {
		wssdCloudAgentVersion = versionPair.WssdCloudAgentVersion
		mocVersion = versionPair.MocVersion
	}

	infoLog := MocInfoLog{
		MocDeploymentId:       deploymentId,
		WssdCloudAgentVersion: wssdCloudAgentVersion,
		MocVersion:            mocVersion,
	}
	jsonData, err := json.Marshal(infoLog)
	if err != nil {
		klog.Error("Unable to serialize moc info log object. ", err)
	} else {
		klog.Info(string(jsonData))
	}
}

func getHealthService(scope scope.ScopeInterface) *health.Service {
	// if healthService instance is created, directy return instance
	if healthService != nil {
		return healthService
	}

	healthService = health.NewService(scope)
	return healthService
}

func getVersionsService(scope scope.ScopeInterface) *versions.Service {
	if versionsService != nil {
		return versionsService
	}

	versionsService = versions.NewService(scope)
	return versionsService
}
