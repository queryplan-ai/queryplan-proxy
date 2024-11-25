package upstream

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/moby/moby/client"
)

func StartMysql(logStdout bool, logStderr bool) (*UpsreamProcess, error) {
	stopAndDelete := make(chan struct{})
	stoppedAndDeleted := make(chan error)

	port := 3000 + rand.Intn(1000)
	containerName := fmt.Sprintf("mysql-%d", port)
	rootPassword := "rootpassword"
	password := "password"

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("error creating Docker client: %v", err)
	}

	ctx := context.Background()

	mysqlImage := "mysql:latest"
	reader, err := cli.ImagePull(ctx, mysqlImage, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("error pulling MySQL image: %v", err)
	}
	defer reader.Close()

	// Consume the pull output to ensure the pull is complete
	_, _ = io.Copy(io.Discard, reader)

	containerConfig := &container.Config{
		Image: mysqlImage,
		Env: []string{
			fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", rootPassword),
			"MYSQL_DATABASE=testdb",
			"MYSQL_USER=testuser",
			fmt.Sprintf("MYSQL_PASSWORD=%s", password),
		},
		ExposedPorts: nat.PortSet{
			"3306/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"3306/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(port),
				},
			},
		},
	}

	networkConfig := &network.NetworkingConfig{}

	containerResp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("error creating MySQL container: %v", err)
	}

	err = cli.ContainerStart(ctx, containerResp.ID, container.StartOptions{})
	if err != nil {
		return nil, fmt.Errorf("error starting MySQL container: %v", err)
	}

	// Health check: Ensure MySQL is ready
	healthCheckCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // 30 seconds timeout for health check
	defer cancel()

	dsn := fmt.Sprintf("testuser:%s@tcp(localhost:%d)/testdb", password, port)
	ready := waitForMySQL(healthCheckCtx, dsn)
	if !ready {
		// Cleanup the container if health check fails
		_ = cli.ContainerStop(ctx, containerResp.ID, container.StopOptions{})
		_ = cli.ContainerRemove(ctx, containerResp.ID, container.RemoveOptions{})
		return nil, fmt.Errorf("mysql container failed health check")
	}

	go func(containerID string) {
		defer close(stoppedAndDeleted)

		<-stopAndDelete

		fmt.Printf("Stopping mysql container %s\n", containerID)
		// Stop the container so that it removes itself
		if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			stoppedAndDeleted <- fmt.Errorf("failed to stop mysql: %v", err)
		}

		if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
			stoppedAndDeleted <- fmt.Errorf("failed to remove mysql: %v", err)
		}

		stoppedAndDeleted <- nil
	}(containerResp.ID)

	return &UpsreamProcess{
		stopAndDelete:     stopAndDelete,
		stoppedAndDeleted: stoppedAndDeleted,
		port:              port,
		password:          password,
	}, nil
}

func waitForMySQL(ctx context.Context, dsn string) bool {
	// this sleep isn't necessary, but it makes the logs quieter
	// becuase mysql seems to take at least 5 seconds to start locally
	time.Sleep(time.Second * 5)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("MySQL health check timed out.")
			return false
		default:
			db, err := sql.Open("mysql", dsn)
			if err == nil {
				defer db.Close()
				if err = db.Ping(); err == nil {
					fmt.Println("MySQL is healthy.")
					return true
				}
			}
			fmt.Println("Waiting for MySQL to become healthy...")
			time.Sleep(1 * time.Second) // Wait before retrying
		}
	}
}
