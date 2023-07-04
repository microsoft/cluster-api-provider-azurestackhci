package azurestackhci

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mocerrors "github.com/microsoft/moc/pkg/errors"
	"k8s.io/klog"

	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc-sdk-for-go/services/admin/deploymentid"
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

func WriteMocOperationLog(scope scope.ScopeInterface, operation MocOperation, crResourceName string, mocResourceType MocResourceType, mocResourceName string, params interface{}, err error) {
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

	WriteMocDeploymentIdLog(scope)
}

func GenerateMocResourceName(nameSegments ...string) string {
	return strings.Join(nameSegments, "/")
}

var (
	deploymentIdClient deploymentid.DeploymentIdClient
)

func WriteMocDeploymentIdLog(scope scope.ScopeInterface) {
	if deploymentIdClient == nil {
		deploymentIdClient = getDeploymentIdClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer())
	}

	deploymentId, err := deploymentIdClient.GetDeploymentId()
	if err != nil {
		klog.Error("Unable to get moc deployment id. ", err)
	} else {
		klog.Info("MOC Deployment Id: %s", deploymentId)
	}
}

// getDeploymentIdClient creates a new deployment id client.
func getDeploymentIdClient(cloudAgentFqdn string, authorizer auth.Authorizer) deploymentid.DeploymentIdClient {
	client, _ := deploymentid.NewDeploymentIdClient(cloudAgentFqdn, authorizer)
	return *client
}
