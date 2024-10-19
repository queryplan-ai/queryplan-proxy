package main

import (
	"context"
	"dagger/queryplan-proxy/internal/dagger"
)

func (q *QueryplanProxy) Validate(ctx context.Context, source *dagger.Directory) *dagger.Container {
	container := buildEnv(ctx, source)
	return container.
		WithExec([]string{"make", "test"})
}
