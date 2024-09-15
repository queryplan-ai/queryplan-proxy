package main

import (
	"context"
	"dagger/queryplan-proxy/internal/dagger"
	"fmt"
)

func (q *QueryplanProxy) Test(ctx context.Context, source *dagger.Directory) *dagger.Container {
	container := q.buildEnv(ctx, source)

	return container.
		WithExec([]string{"make", "test"})
}

func (q *QueryplanProxy) Build(ctx context.Context, source *dagger.Directory) *dagger.Container {
	container := q.buildEnv(ctx, source)

	return container.
		WithExec([]string{"make", "build"})
}

// Publish publishes the QueryplanProxy to the GitHub Container Registry.
//
// Example to publish a tagged release and latest:
//
//	dagger call publish --source . --username=marccampbell --token env:GITHUB_API_TOKEN --latest=true --tag=0.0.1
func (q *QueryplanProxy) Publish(
	ctx context.Context,
	source *dagger.Directory,
	username string,
	token *dagger.Secret,
	// +optional
	tag string,
	// +optional
	// +default=true
	latest bool,
) ([]string, error) {
	container := q.buildEnv(ctx, source)

	binary := container.
		WithExec([]string{"make", "build"}).
		File("/go/src/github.com/queryplan-ai/queryplan-proxy/bin/queryplan-proxy")

	// build the release container with the binary in it
	releaseContainer := q.releaseEnv(ctx, binary)

	releases := []string{}

	if tag != "" {
		tagRelease, err := releaseContainer.
			WithRegistryAuth("ghcr.io", username, token).
			Publish(ctx, fmt.Sprintf("ghcr.io/queryplan-ai/queryplan-proxy:%s", tag))
		if err != nil {
			return nil, err
		}
		releases = append(releases, tagRelease)
	}

	if latest {
		latestRelease, err := releaseContainer.
			WithRegistryAuth("ghcr.io", username, token).
			Publish(ctx, "ghcr.io/queryplan-ai/queryplan-proxy:latest")
		if err != nil {
			return nil, err
		}
		releases = append(releases, latestRelease)
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
		Container().
		WithFile("/queryplan-proxy", binaryFile)

	return releaseContainer
}
