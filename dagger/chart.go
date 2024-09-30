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

	return chart
}
