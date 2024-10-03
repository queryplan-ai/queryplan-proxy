package main

import (
	"context"
	"dagger/queryplan-proxy/internal/dagger"
	"fmt"
)

func buildChart(ctx context.Context, source *dagger.Directory, version string) *dagger.Container {
	container := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("alpine/helm:3.16.1")

	chartSource := source.Directory("chart/queryplan-proxy")
	container = container.WithDirectory("/chart", chartSource)

	// replace the version and app version in the chart
	chart := container.WithExec([]string{"sed", "-i", fmt.Sprintf("s/version: .*/version: %s/", version), "/chart/Chart.yaml"})
	chart = chart.WithExec([]string{"sed", "-i", fmt.Sprintf("s/appVersion: .*/appVersion: %s/", version), "/chart/Chart.yaml"})

	// replace the tag in in the values.yaml
	chart = chart.WithExec([]string{"sed", "-i", fmt.Sprintf("s/tag: .*/tag: %s/", version), "/chart/values.yaml"})

	return chart
}

func publishChart(ctx context.Context, chartArchive *dagger.File, version string, username string, token *dagger.Secret) error {
	container := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("alpine/helm:3.16.1")

	// add the chart to the container
	container = container.WithFile("/chart.tgz", chartArchive)

	tokenPlaintext, _ := token.Plaintext(ctx)

	// exec to log in to the oci registry and publish
	c := container.
		WithExec([]string{
			"helm",
			"registry",
			"login",
			"ghcr.io",
			"--username",
			username,
			"--password",
			tokenPlaintext,
		}).
		WithExec([]string{
			"helm",
			"push",
			"/chart.tgz",
			"oci://ghcr.io/queryplan-ai",
		})

	stdout, err := c.Stdout(ctx)
	if err != nil {
		return err
	}

	stderr, err := c.Stderr(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("stdout: %s\n", stdout)
	fmt.Printf("stderr: %s\n", stderr)

	return nil
}
