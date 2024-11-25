package upstream

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/rand"
)

func StartPostgres(logStdout bool, logStderr bool) (*UpsreamProcess, error) {
	stopAndDelete := make(chan struct{})
	stoppedAndDeleted := make(chan error)

	port := 3000 + rand.Intn(1000)
	containerName := fmt.Sprintf("postgres-%d", port)
	password := "password"

	cmd := exec.Command("docker", "run",
		"--name", containerName,
		"--rm",
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		"-p", fmt.Sprintf("%d:5432", port),
		"postgres:15",
	)

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres: %v", err)
	}

	// apply a schema
	connectionURI := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres", "postgres", password, "localhost", port)
	conn, err := pgx.Connect(context.Background(), connectionURI)
	if err != nil {
		return nil, err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "create table users (id int, name varchar(255))")
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(context.Background(), "insert into users (id, name) values (1, 'John')")
	if err != nil {
		return nil, err
	}

	conn.Close(context.Background())

	return &UpsreamProcess{
		stopAndDelete:     stopAndDelete,
		stoppedAndDeleted: stoppedAndDeleted,
		port:              port,
		password:          password,
	}, nil
}
