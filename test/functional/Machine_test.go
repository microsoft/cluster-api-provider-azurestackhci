package functional

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/microsoft/cluster-api-provider-azurestackhci/test/mockcloudagent/pkg/agents"
	mocclient "github.com/microsoft/moc-sdk-for-go/pkg/client"
	compute_pb "github.com/microsoft/moc/rpc/cloudagent/compute"
	network_pb "github.com/microsoft/moc/rpc/cloudagent/network"
	"google.golang.org/grpc"
)

const (
	ManagementClusterName = "k3d-testmgmtcluster"
)

func TestBasic(t *testing.T) {
	defer func() {
		err := runCommand(t, "bash", "./Scripts/ClusterCleanup.sh")
		if err != nil {
			t.Errorf("ClusterCleanup.sh failed: %v", err)
		}
	}()

	err := runCommand(t, "bash", "./Scripts/Build.sh")
	if err != nil {
		t.Fatalf("Build.sh failed: %v", err)
	}

	err = runCommand(t, "bash", "./Scripts/ClusterSetup.sh")
	if err != nil {
		t.Fatalf("ClusterSetup.sh failed: %v", err)
	}

	hostIPAddr, err := getDockerGatewayIPAddr(ManagementClusterName)
	if err != nil {
		t.Fatalf("getDockerGatewayIPAddr failed: %v", err)
	}

	mockCloudAgentAddr := fmt.Sprintf("%s:%d", hostIPAddr, mocclient.ServerPort)
	t.Logf("Mock Cloud Agent: %s", mockCloudAgentAddr)

	listen, err := net.Listen("tcp", mockCloudAgentAddr)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	compute_pb.RegisterVirtualMachineAgentServer(grpcServer, &agents.VirtualMachineAgentServer{})
	network_pb.RegisterVirtualNetworkAgentServer(grpcServer, &agents.VirtualNetworkAgentServer{})

	serveFinished := make(chan bool)
	var serveError error
	go func() {
		serveError = grpcServer.Serve(listen)
		serveFinished <- true
	}()

	err = runCommand(t, "bash", "./Scripts/CAPISetup.sh")
	if err != nil {
		t.Fatalf("CAPISetup.sh failed: %v", err)
	}

	err = runCommand(t, "bash", "./Scripts/WorkerClusterSetup.sh")
	if err != nil {
		t.Fatalf("WorkerClusterSetup.sh failed: %v", err)
	}

	time.Sleep(time.Second * 5)

	grpcServer.GracefulStop()
	<-serveFinished

	if serveError != nil {
		t.Errorf("Serve failed: %v", err)
	}

	t.Logf("Finished. Time: %v", time.Now())
}

func getDockerGatewayIPAddr(networkName string) (string, error) {
	dockerClient, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return "", err
	}
	defer dockerClient.Close()

	networkInfo, err := dockerClient.NetworkInspect(context.Background(), networkName, dockertypes.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}

	return networkInfo.IPAM.Config[0].Gateway, nil
}

func runCommand(t *testing.T, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	fmt.Println(cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Log(err)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Log(err)
		return err
	}

	err = cmd.Start()
	if err != nil {
		t.Log(err)
		return err
	}

	go handleLines(stdout, func(str string) { fmt.Println("    ", str) })
	go handleLines(stderr, func(str string) { fmt.Println("    ", str) })

	err = cmd.Wait()
	if err != nil {
		t.Log(err)
		return err
	}
	return nil
}

func handleLines(src io.Reader, lineFunc func(str string)) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		lineFunc(scanner.Text()) // Println will add back the final '\n'
	}
}
