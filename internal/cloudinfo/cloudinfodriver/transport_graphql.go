// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudinfodriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/99designs/gqlgen/handler"
	"github.com/go-kit/kit/endpoint"

	"github.com/banzaicloud/cloudinfo/.gen/api/graphql"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
)

// MakeGraphQLHandler mounts all of the service endpoints into a GraphQL handler.
func MakeGraphQLHandler(
	endpoints Endpoints,
	providerEndpoints ProviderEndpoints,
	serviceEndpoints ServiceEndpoints,
	regionEndpoints RegionEndpoints,
	errorHandler cloudinfo.ErrorHandler,
) http.Handler {
	return handler.GraphQL(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: &resolver{
			endpoints:         endpoints,
			providerEndpoints: providerEndpoints,
			serviceEndpoints:  serviceEndpoints,
			regionEndpoints:   regionEndpoints,
			errorHandler:      errorHandler,
		},
	}))
}

type resolver struct {
	endpoints         Endpoints
	providerEndpoints ProviderEndpoints
	serviceEndpoints  ServiceEndpoints
	regionEndpoints   RegionEndpoints
	errorHandler      cloudinfo.ErrorHandler
}

func (r *resolver) Query() graphql.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *resolver }

func (r *queryResolver) Providers(ctx context.Context) ([]cloudinfo.Provider, error) {
	resp, err := r.providerEndpoints.List(ctx, nil)
	if err != nil {
		r.errorHandler.Handle(err)

		return nil, errors.New("internal server error")
	}

	if f, ok := resp.(endpoint.Failer); ok && f.Failed() != nil {
		return nil, f.Failed()
	}

	return resp.(listProvidersResponse).Providers, nil
}

func (r *queryResolver) InstanceTypes(ctx context.Context, provider string, service string, region *string, zone *string, filter *cloudinfo.InstanceTypeQueryFilter) ([]cloudinfo.InstanceType, error) {
	req := instanceTypeQueryRequest{
		Provider: provider,
		Service:  service,
		Region:   region,
		Zone:     zone,
		Filter:   filter,
	}

	resp, err := r.endpoints.InstanceTypeQuery(ctx, req)
	if err != nil {
		r.errorHandler.Handle(err)

		return nil, errors.New("internal server error")
	}

	if f, ok := resp.(endpoint.Failer); ok && f.Failed() != nil {
		return nil, f.Failed()
	}

	return resp.(instanceTypeQueryResponse).InstanceTypes, nil
}

func (r *resolver) Provider() graphql.ProviderResolver {
	return &providerResolver{r}
}

type providerResolver struct{ *resolver }

func (r *providerResolver) Services(ctx context.Context, obj *cloudinfo.Provider) ([]cloudinfo.Service, error) {
	req := listServicesRequest{
		Provider: obj.Name,
	}

	resp, err := r.serviceEndpoints.List(ctx, req)
	if err != nil {
		r.errorHandler.Handle(err)

		return nil, errors.New("internal server error")
	}

	if f, ok := resp.(endpoint.Failer); ok && f.Failed() != nil {
		return nil, f.Failed()
	}

	return resp.(listServicesResponse).Services, nil
}

func (r *resolver) Service() graphql.ServiceResolver {
	return &serviceResolver{r}
}

type serviceResolver struct{ *resolver }

func (r *serviceResolver) Regions(ctx context.Context, obj *cloudinfo.Service) ([]cloudinfo.Region, error) {
	req := listRegionsRequest{
		Provider: obj.ProviderName(),
		Service:  obj.Name,
	}

	resp, err := r.regionEndpoints.List(ctx, req)
	if err != nil {
		r.errorHandler.Handle(err)

		return nil, errors.New("internal server error")
	}

	if f, ok := resp.(endpoint.Failer); ok && f.Failed() != nil {
		return nil, f.Failed()
	}

	return resp.(listRegionsResponse).Regions, nil
}
