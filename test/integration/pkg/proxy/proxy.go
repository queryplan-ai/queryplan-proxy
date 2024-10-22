package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

type ProxyProcess struct {
	cmd  *exec.Cmd
	done chan error
}

func StartProxy(binaryPath string, args ...string) (*ProxyProcess, error) {
	cmd := exec.Command(binaryPath, args...)

	// Redirect stdout and stderr to our process
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy: %v", err)
	}

	proxy := &ProxyProcess{
		cmd:  cmd,
		done: make(chan error, 1),
	}

	// Wait for the process in a goroutine
	go func() {
		proxy.done <- cmd.Wait()
	}()

	// Give the process a moment to start up
	time.Sleep(time.Second)

	return proxy, nil
}

func (p *ProxyProcess) Stop() error {
	if p.cmd.Process != nil {
		err := p.cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to kill process: %v", err)
		}
	}

	// Wait for the process to exit
	select {
	case err := <-p.done:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for process to exit")
	}
}
