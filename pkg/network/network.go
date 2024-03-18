package network

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func EndpointChecker(ip string) healthz.Checker {
	return func(req *http.Request) error {
		var lastErr error

		backoff := wait.Backoff{
			Duration: 250 * time.Millisecond,
			Factor:   1.5,
			Steps:    9,
			Jitter:   0.1,
		}
		defaultTimeout := 1 * time.Second

		err := wait.ExponentialBackoff(backoff, func() (bool, error) {
			_, err := net.DialTimeout("tcp", ip, defaultTimeout)
			if err != nil {
				lastErr = fmt.Errorf("failed to dial: %v", err)
				return false, nil
			}

			return true, nil
		})
		if err != nil {
			return lastErr
		}

		return nil
	}
}
