package main

import (
	"context"
	"dagger/queryplan-proxy/internal/dagger"
)

func (q *QueryplanProxy) Build(ctx context.Context, source *dagger.Directory) *dagger.Container {
	container := q.buildEnv(ctx, source)

	return container.
		WithExec([]string{"make", "build"})
}
