package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/microsoft/cluster-api-provider-azurestackhci/test/mockcloudagent/pkg/agents"
	mocclient "github.com/microsoft/moc-sdk-for-go/pkg/client"
	compute_pb "github.com/microsoft/moc/rpc/cloudagent/compute"
	network_pb "github.com/microsoft/moc/rpc/cloudagent/network"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func main() {
	var ipAddress, dockerNetwork string
	allErrorsCmd := &cobra.Command{
		Use:   "errors",
		Short: "Respond with errors to all service requests",
		Run: func(cmd *cobra.Command, args []string) {
			runErr := runAllErrors(cmd, args, ipAddress, dockerNetwork)
			if runErr != nil {
				fmt.Printf("%v\n", runErr)
			}
		},
	}

	rootCmd := &cobra.Command{
		Use: "mockcloudagent",
	}
	rootCmd.PersistentFlags().StringVarP(&ipAddress, "ip-address", "a", "127.0.0.1", "IP address (and port) to listen on.")
	rootCmd.PersistentFlags().StringVarP(&dockerNetwork, "docker-network", "d", "", "Docker network to listen on.")
	rootCmd.AddCommand(allErrorsCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func runAllErrors(cmd *cobra.Command, args []string, ipAddress string, dockerNetwork string) error {
	var err error
	if dockerNetwork != "" {
		ipAddress, err = getDockerGatewayIPAddr(dockerNetwork)
		if err != nil {
			return err
		}
	}

	// Open network port.
	addr := fmt.Sprintf("%s:%d", ipAddress, mocclient.ServerPort)
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	fmt.Printf("Listening on: %s\n", listen.Addr())

	// Setup gRPC server.
	grpcServer := grpc.NewServer()
	compute_pb.RegisterVirtualMachineAgentServer(grpcServer, &agents.VirtualMachineAgentServer{})
	network_pb.RegisterVirtualNetworkAgentServer(grpcServer, &agents.VirtualNetworkAgentServer{})

	// Listen for Ctrl+C signal.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// Request gRPC server to stop.
			grpcServer.GracefulStop()
			fmt.Println()
		}
	}()

	// Run gRPC server.
	err = grpcServer.Serve(listen)
	if err != nil {
		return err
	}

	return nil
}

// Get the gateway address of a docker network.
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
