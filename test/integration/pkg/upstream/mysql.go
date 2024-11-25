package upstream

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
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
		log.Fatalf("Error creating Docker client: %v", err)
	}

	ctx := context.Background()

	mysqlImage := "mysql:9.1"
	reader, err := cli.ImagePull(ctx, mysqlImage, image.PullOptions{})
	if err != nil {
		log.Fatalf("Error pulling MySQL image: %v", err)
	}
	defer reader.Close()

	// Optionally: Consume the pull output to ensure the pull is complete
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
		log.Fatalf("Error creating MySQL container: %v", err)
	}

	err = cli.ContainerStart(ctx, containerResp.ID, container.StartOptions{})
	if err != nil {
		log.Fatalf("Error starting MySQL container: %v", err)
	}

	go func(containerID string) {
		defer close(stoppedAndDeleted)

		<-stopAndDelete

		fmt.Printf("Stopping mysql container %s\n", containerID)
		// stop the container so that it rm itself
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
