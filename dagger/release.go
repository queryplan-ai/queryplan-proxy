package main

import (
	"context"
	"dagger/queryplan-proxy/internal/dagger"
	"fmt"
)

func (q *QueryplanProxy) Release(
	ctx context.Context,

	// +defaultPath="/"
	source *dagger.Directory,

	version string,
	username string,
	token *dagger.Secret,
) ([]string, error) {
	container := q.buildEnv(ctx, source)

	binary := container.
		WithExec([]string{"make", "build"}).
		File("/go/src/github.com/queryplan-ai/queryplan-proxy/bin/queryplan-proxy")

	// build the release container with the binary in it
	releaseContainer := q.releaseEnv(ctx, binary)
	releaseContainer = releaseContainer.WithEntrypoint([]string{
		"/queryplan-proxy",
	})
	releaseContainer = releaseContainer.WithDefaultArgs([]string{
		"start",
	})

	releases := []string{}

	tagRelease, err := releaseContainer.
		WithRegistryAuth("ghcr.io", username, token).
		Publish(ctx, fmt.Sprintf("ghcr.io/queryplan-ai/queryplan-proxy:%s", version))
	if err != nil {
		return nil, err
	}
	releases = append(releases, tagRelease)

	latestRelease, err := releaseContainer.
		WithRegistryAuth("ghcr.io", username, token).
		Publish(ctx, "ghcr.io/queryplan-ai/queryplan-proxy:latest")
	if err != nil {
		return nil, err
	}
	releases = append(releases, latestRelease)

	chart := buildChart(ctx, source, version)
	chartArchive := chart.
		WithExec([]string{"helm", "package", "/chart"}).
		File(fmt.Sprintf("/apps/queryplan-proxy-chart-%s.tgz", version))
	if err != nil {
		return nil, err
	}
	err = publishChart(ctx, chartArchive, version, username, token)
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (q *QueryplanProxy) buildEnv(ctx context.Context, source *dagger.Directory) *dagger.Container {
	// exclude some directories
	source = source.WithoutDirectory("dagger")
	source = source.WithoutDirectory("okteto")

	cache := dag.CacheVolume("queryplan-proxy")

	buildContainer := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("golang:1.23")

	return buildContainer.
		WithDirectory("/go/src/github.com/queryplan-ai/queryplan-proxy", source).
		WithWorkdir("/go/src/github.com/queryplan-ai/queryplan-proxy").
		WithMountedCache("/go/pkg/mod", cache).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/build-cache", cache).
		WithEnvVariable("GOCACHE", "/go/build-cache").
		WithExec([]string{"go", "mod", "download"})
}

func (q *QueryplanProxy) releaseEnv(ctx context.Context, binaryFile *dagger.File) *dagger.Container {
	releaseContainer := dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Arch: "amd64",
		}).
		WithFile("/queryplan-proxy", binaryFile)

	return releaseContainer
}
