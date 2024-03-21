package network

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func EndpointChecker(targetIPAddress string,
	initialBackoffDuration time.Duration,
	backoffFactor float64,
	maxRetryAttempts int,
	backoffJitterFactor float64,
	dialTimeout time.Duration) healthz.Checker {

	return func(req *http.Request) error {
		var lastErr error

		backoff := wait.Backoff{
			Duration: initialBackoffDuration,
			Factor:   backoffFactor,
			Steps:    maxRetryAttempts,
			Jitter:   backoffJitterFactor,
		}

		err := wait.ExponentialBackoff(backoff, func() (bool, error) {
			_, err := net.DialTimeout("tcp", targetIPAddress, dialTimeout)
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
