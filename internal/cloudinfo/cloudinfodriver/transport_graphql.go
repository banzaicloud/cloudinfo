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
func MakeGraphQLHandler(endpoints Endpoints, errorHandler cloudinfo.ErrorHandler) http.Handler {
	return handler.GraphQL(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: &resolver{
			endpoints:    endpoints,
			errorHandler: errorHandler,
		},
	}))
}

type resolver struct {
	endpoints    Endpoints
	errorHandler cloudinfo.ErrorHandler
}

func (r *resolver) Query() graphql.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *resolver }

func (r *queryResolver) InstanceTypes(ctx context.Context, provider string, service string, region *string, zone *string, filter cloudinfo.InstanceTypeQueryFilter) ([]cloudinfo.InstanceType, error) {
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
