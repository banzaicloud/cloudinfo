package api

import (
	"context"

	"github.com/banzaicloud/cloudinfo/.gen/api/graphql"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct{}

func (r *Resolver) Query() search.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) InstanceTypes(ctx context.Context, provider string, region *string, zone *string, filter search.InstanceTypeQueryInput) ([]search.InstanceType, error) {
	return nil, nil
}
