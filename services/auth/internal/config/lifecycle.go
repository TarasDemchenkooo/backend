package config

import (
	"fmt"
	"time"
)

type Startup struct {
	Timeout time.Duration
}

func loadStartup() (Startup, error) {
	timeout, err := envDuration("STARTUP_TIMEOUT", 15*time.Second)
	if err != nil {
		return Startup{}, err
	}

	if timeout <= 0 {
		return Startup{}, fmt.Errorf("env STARTUP_TIMEOUT must be positive, got %s", timeout)
	}

	return Startup{Timeout: timeout}, nil
}

type Shutdown struct {
	DrainDelay time.Duration
	Timeout    time.Duration
}

func loadShutdown() (Shutdown, error) {
	drainDelay, err := envDuration("SHUTDOWN_DRAIN_DELAY", 5*time.Second)
	if err != nil {
		return Shutdown{}, err
	}

	if drainDelay < 0 {
		return Shutdown{}, fmt.Errorf("env SHUTDOWN_DRAIN_DELAY must not be negative, got %s", drainDelay)
	}

	timeout, err := envDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Shutdown{}, err
	}

	if timeout <= 0 {
		return Shutdown{}, fmt.Errorf("env SHUTDOWN_TIMEOUT must be positive, got %s", timeout)
	}

	return Shutdown{DrainDelay: drainDelay, Timeout: timeout}, nil
}
