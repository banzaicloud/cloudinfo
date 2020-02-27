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

	"emperror.dev/errors"
	"github.com/go-kit/kit/endpoint"
	kitoc "github.com/go-kit/kit/tracing/opencensus"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

// RegionEndpoints collects all of the endpoints that compose a region service.
// It's meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type RegionEndpoints struct {
	ListRegions endpoint.Endpoint
	ListZones   endpoint.Endpoint
}

// MakeRegionEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeRegionEndpoints(s RegionService, logger cloudinfo.Logger) RegionEndpoints {
	return RegionEndpoints{
		ListRegions: endpoint.Chain(
			kitoc.TraceEndpoint(OperationRegionListRegions),
			LogEndpoint(OperationRegionListRegions, logger),
		)(MakeListRegionsEndpoint(s)),
		ListZones: endpoint.Chain(
			kitoc.TraceEndpoint(OperationRegionListZones),
			LogEndpoint(OperationRegionListZones, logger),
		)(MakeListZonesEndpoint(s)),
	}
}

type listRegionsRequest struct {
	Provider string
	Service  string
}

type listRegionsResponse struct {
	Regions []cloudinfo.Region
	Err     error
}

func (r listRegionsResponse) Failed() error {
	return r.Err
}

// MakeListRegionsEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListRegionsEndpoint(s RegionService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(listRegionsRequest)

		regions, err := s.ListRegions(ctx, req.Provider, req.Service)

		if err != nil {
			if b, ok := errors.Cause(err).(businessError); ok && b.IsBusinessError() {
				return listProvidersResponse{
					Err: err,
				}, nil
			}

			return nil, err
		}

		resp := listRegionsResponse{
			Regions: regions,
		}

		return resp, nil
	}
}

type listZonesRequest struct {
	Provider string
	Service  string
	Region   string
}

type listZonesResponse struct {
	Zones []cloudinfo.Zone
	Err   error
}

func (r listZonesResponse) Failed() error {
	return r.Err
}

// MakeListZonesEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListZonesEndpoint(s RegionService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(listZonesRequest)

		zones, err := s.ListZones(ctx, req.Provider, req.Service, req.Region)

		if err != nil {
			if b, ok := errors.Cause(err).(businessError); ok && b.IsBusinessError() {
				return listProvidersResponse{
					Err: err,
				}, nil
			}

			return nil, err
		}

		resp := listZonesResponse{
			Zones: zones,
		}

		return resp, nil
	}
}
