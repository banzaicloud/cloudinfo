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

	"github.com/go-kit/kit/endpoint"
	kitoc "github.com/go-kit/kit/tracing/opencensus"
	"github.com/pkg/errors"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
)

// Endpoints collects all of the endpoints that compose a service service.
// It's meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type ProviderEndpoints struct {
	List endpoint.Endpoint
}

// MakeProviderEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeProviderEndpoints(s ProviderService, logger cloudinfo.Logger) ProviderEndpoints {
	return ProviderEndpoints{
		List: endpoint.Chain(
			kitoc.TraceEndpoint(OperationProviderListProviders),
			LogEndpoint(OperationProviderListProviders, logger),
		)(MakeListProvidersEndpoint(s)),
	}
}

type listProvidersResponse struct {
	Providers []cloudinfo.Provider
	Err       error
}

func (r listProvidersResponse) Failed() error {
	return r.Err
}

// MakeListProvidersEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListProvidersEndpoint(s ProviderService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		providers, err := s.ListProviders(ctx)

		if err != nil {
			if b, ok := errors.Cause(err).(businessError); ok && b.IsBusinessError() {
				return listProvidersResponse{
					Err: err,
				}, nil
			}

			return nil, err
		}

		resp := listProvidersResponse{
			Providers: providers,
		}

		return resp, nil
	}
}
