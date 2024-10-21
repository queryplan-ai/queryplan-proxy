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
	githubToken *dagger.Secret,
) error {
	latestVersion, newVersion, err := determineVersions(ctx, version)
	if err != nil {
		return err
	}

	fmt.Printf("Releasing %s -> %s\n", latestVersion, newVersion)

	// push the tag
	githubTokenPlaintext, err := githubToken.Plaintext(ctx)
	if err != nil {
		return err
	}
	tagContainer := dag.Container().
		From("alpine/git:latest").
		WithMountedDirectory("/go/src/github.com/queryplan-ai/queryplan-proxy", source).
		WithWorkdir("/go/src/github.com/queryplan-ai/queryplan-proxy").
		WithExec([]string{"git", "remote", "add", "tag", fmt.Sprintf("https://%s@github.com/queryplan-ai/queryplan-proxy.git", githubTokenPlaintext)}).
		WithExec([]string{"git", "tag", newVersion}).
		WithExec([]string{"git", "push", "tag", newVersion})
	_, err = tagContainer.Stdout(ctx)
	if err != nil {
		return err
	}

	// store all release assets for the github release
	releaseAssets := []*dagger.File{}

	// update the source with the tag so that our git history later has it
	source = tagContainer.Directory("/go/src/github.com/queryplan-ai/queryplan-proxy")

	container := buildEnv(ctx, source)
	binary := container.
		WithExec([]string{"make", "build"}).
		File("/go/src/github.com/queryplan-ai/queryplan-proxy/bin/queryplan-proxy")
	releaseAssets = append(releaseAssets, binary)

	releaseContainer := dag.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("debian:bullseye-slim").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "ca-certificates"}).
		WithExec([]string{"rm", "-rf", "/var/lib/apt/lists/*"}).
		WithFile("/queryplan-proxy", binary)
	releaseContainer = releaseContainer.WithEntrypoint([]string{
		"/queryplan-proxy",
	}).WithDefaultArgs([]string{
		"start",
	})

	_, err = releaseContainer.
		WithRegistryAuth("ghcr.io", username, githubToken).
		Publish(ctx, fmt.Sprintf("ghcr.io/queryplan-ai/queryplan-proxy:%s", newVersion))
	if err != nil {
		return err
	}

	_, err = releaseContainer.
		WithRegistryAuth("ghcr.io", username, githubToken).
		Publish(ctx, "ghcr.io/queryplan-ai/queryplan-proxy:latest")
	if err != nil {
		return err
	}

	chartContainer := buildChart(ctx, source, newVersion)
	chartArchive := chartContainer.
		WithExec([]string{"helm", "package", "/chart"}).
		File(fmt.Sprintf("/apps/queryplan-proxy-chart-%s.tgz", newVersion))
	releaseAssets = append(releaseAssets, chartArchive)

	err = publishChart(ctx, chartArchive, newVersion, username, githubToken)
	if err != nil {
		return err
	}

	// create a release on github
	if err := dag.Gh().
		WithToken(githubToken).
		WithRepo("queryplan-ai/queryplan-proxy").
		WithSource(source).
		Release().
		Create(ctx, newVersion, newVersion, dagger.GhReleaseCreateOpts{
			Files: releaseAssets,
		}); err != nil {
		return err
	}

	return nil
}

func buildEnv(ctx context.Context, source *dagger.Directory) *dagger.Container {
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
