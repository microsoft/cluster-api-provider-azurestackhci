package azurestackhci

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mocerrors "github.com/microsoft/moc/pkg/errors"
	"k8s.io/klog"

	"github.com/microsoft/moc-sdk-for-go/services/admin/health"
	"github.com/microsoft/moc/pkg/auth"
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

var healthClient *health.HealthClient

func WriteMocDeploymentIdLog(ctx context.Context, cloudAgentFqdn string, authorizer auth.Authorizer) {
	deploymentId, err := getHealthClient(cloudAgentFqdn, authorizer).GetDeploymentId(ctx)
	if err != nil {
		klog.Error("Unable to get moc deployment id. ", err)
	} else {
		klog.Infof("MOC Deployment Id: %s", deploymentId)
	}
}

func getHealthClient(cloudAgentFqdn string, authorizer auth.Authorizer) *health.HealthClient {
	// if deploymentIdClient instance is created, directy return instance
	if healthClient != nil {
		return healthClient
	}

	client, err := health.NewHealthClient(cloudAgentFqdn, authorizer)
	if err != nil {
		klog.Error("Unable to create health client. ", err)
		return nil
	}
	healthClient = client
	return healthClient
}
