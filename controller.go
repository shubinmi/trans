package trans

import (
	"context"
)

type ControllerEndpoint func(ctx context.Context, params *Params) (Result, error)

type ControllerMiddleware func(next ControllerEndpoint) ControllerEndpoint

type opt struct {
	ctrlHandler ControllerEndpoint
}

func ControllerDecorate(endpoint ControllerEndpoint, decorators ...ControllerMiddleware) ControllerEndpoint {
	index := len(decorators) - 1
	for i := range decorators {
		endpoint = decorators[index-i](endpoint)
	}
	return endpoint
}
