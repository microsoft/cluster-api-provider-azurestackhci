package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/health"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/versions"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

type ResourceType string

const (
	CRD    ResourceType = "CRD"
	Status ResourceType = "Status"
)

type Operation string

const (
	Create         Operation = "Create"
	CreateOrUpdate Operation = "CreateOrUpdate"
	Update         Operation = "Update"
	Delete         Operation = "Delete"
	Get            Operation = "Get"
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

func WriteMocOperationLog(logger logr.Logger, operation Operation, crResourceName string, mocResourceType MocResourceType, mocResourceName string, params interface{}, err error) {
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
		logger.Error(err, "Unable to serialize operation log object.", "resourceName", crResourceName)
	} else {
		logger.Info(string(jsonData))
	}
}

// RecordHybridAKSCRDChange need to be called when CRD changed.
func RecordHybridAKSCRDChange(logger logr.Logger, parentResource string, resource string, action Operation, resourceType ResourceType, params interface{}, err error) {
	errMessage := ""
	errCode := "0"
	if err != nil {
		errCode = "-1"
		errMessage = err.Error()
	}

	oplog := OperationLog{
		Timestamp:      time.Now().Format(time.RFC3339),
		ParentResource: parentResource,
		Resource:       resource,
		FilterKeyword:  "RESOURCE_ACTION",
		Action:         fmt.Sprintf("%s %s", action, resourceType),
		Params:         params,
		ErrorCode:      errCode,
		Message:        errMessage,
	}

	jsonData, err := json.Marshal(oplog)
	if err != nil {
		logger.Error(err, "Unable to serialize operation log object",
			"timestamp", time.Now().Format(time.RFC3339),
			"parent_resource", parentResource,
			"resource", resource,
			"filter_keyword", "RESOURCE_ACTION",
			"action", action)
	} else {
		logger.Info(string(jsonData))
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
	logger := scope.GetLogger()
	if err != nil {
		logger.Error(err, "Unable to get moc version.")
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
		logger.Error(err, "Unable to serialize moc info log object.")
	} else {
		logger.Info(string(jsonData))
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

func IsCRDUpdate(operationResult ctrlutil.OperationResult) bool {
	if operationResult == ctrlutil.OperationResultCreated || operationResult == ctrlutil.OperationResultUpdatedStatus ||
		operationResult == ctrlutil.OperationResultUpdatedStatusOnly {
		return true
	}

	return false
}

func ConvertOperationResult(operationResult ctrlutil.OperationResult) (Operation, ResourceType) {
	switch operationResult {
	case ctrlutil.OperationResultCreated:
		return Create, CRD
	case ctrlutil.OperationResultUpdated:
		return Update, CRD
	case ctrlutil.OperationResultUpdatedStatus:
		fallthrough
	case ctrlutil.OperationResultUpdatedStatusOnly:
		return Update, Status
	default:
		return "", ""
	}
}
