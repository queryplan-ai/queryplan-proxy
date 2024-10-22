package upstream

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/rand"
)

type UpsreamProcess struct {
	done     chan error
	port     int
	password string
}

func (p *UpsreamProcess) Stop() {
	close(p.done)
}

func (p *UpsreamProcess) Port() int {
	return p.port
}

func StartPostgres(logStdout bool, logStderr bool) (*UpsreamProcess, error) {
	done := make(chan error)
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

	if logStdout {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %v", err)
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Printf("stdout: %s\n", scanner.Text())
			}
			done <- scanner.Err()
		}()
	}

	if logStderr {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stderr pipe: %v", err)
		}

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				fmt.Printf("stderr: %s\n", scanner.Text())
			}
			done <- scanner.Err()
		}()
	}

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres: %v", err)
	}

	go func() {
		<-done

		// stop the container so that it rm itself
		cmd := exec.Command("docker", "stop", containerName)
		err := cmd.Run()
		if err != nil {
			done <- fmt.Errorf("failed to stop postgres: %v", err)
		}
	}()

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
		done:     done,
		port:     port,
		password: password,
	}, nil
}
