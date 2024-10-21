package main

import (
	"context"
	"dagger/queryplan-proxy/internal/dagger"
	"fmt"
)

func (q *QueryplanProxy) Validate(
	ctx context.Context,

	// +defaultPath="/"
	source *dagger.Directory,
) (
	*dagger.File,
	error,
) {
	container := buildEnv(ctx, source)
	if _, err := container.
		WithExec([]string{"make", "test"}).
		Sync(ctx); err != nil {
		return nil, err
	}

	chartContainer := buildChart(ctx, source, "0.0.1-test")
	chartArchive := chartContainer.
		WithExec([]string{"helm", "package", "/chart"}).
		File(fmt.Sprintf("/apps/queryplan-proxy-chart-0.0.1-test.tgz"))

	templateContainer := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("alpine/helm:3.16.1").
		WithFile("/chart.tgz", chartArchive).
		WithExec([]string{"helm", "template", "queryplany", "/chart.tgz",
			"--set", "engine=postgres",
			"--set", "connection.liveUri=postgresql://test:test@postgres:5432/test",
			"--set", "connection.databaseName=test",
			"--set", "connection.bindPort=5432",
			"--set", "connection.upstreamAddress=postgres",
			"--set", "connection.upstreamPort=5432",
			"--set", "connection.token=a-token",
			"--set", "environment=dev",
		})
	stdout, err := templateContainer.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	stderr, err := templateContainer.Stderr(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Printf("stdout: %s\n", stdout)
	fmt.Printf("stderr: %s\n", stderr)

	return chartArchive, nil
}
