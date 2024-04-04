package network

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func EndpointChecker(targetIPAddress string) healthz.Checker {
	return func(req *http.Request) error {
		dialTimeout := 1 * time.Second

		_, err := net.DialTimeout("tcp", targetIPAddress, dialTimeout)
		if err != nil {
			return fmt.Errorf("failed to dial: %v", err)
		}

		return nil
	}
}
