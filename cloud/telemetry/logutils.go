package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/health"
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

var healthService *health.Service

func WriteMocDeploymentIdLog(ctx context.Context, scope scope.ScopeInterface) {
	deploymentId := getHealthService(scope).GetMocDeploymentId(ctx)
	klog.Infof("MOC Deployment Id: %s", deploymentId)
}

func getHealthService(scope scope.ScopeInterface) *health.Service {
	// if healthService instance is created, directy return instance
	if healthService != nil {
		return healthService
	}

	healthService = health.NewService(scope)
	return healthService
}
