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

	"github.com/banzaicloud/cloudinfo/.gen/api/graphql"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
)

// InstanceTypeService filters instance types according to the received query.
type InstanceTypeService interface {
	// Query processes an instance type query and responds with a list match of instance types matching that query.
	Query(ctx context.Context, provider string, service string, query cloudinfo.InstanceTypeQuery) ([]graphql.InstanceType, error)
}

const (
	codeInvalidInstanceTypeQuery int = 1
)

type instanceTypeError struct {
	msg  string
	code int
}

func (e *instanceTypeError) Error() string {
	return e.msg
}

func (e *instanceTypeError) Code() int {
	return e.code
}

// Endpoints collects all of the endpoints that compose an instance type service.
// It's meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	InstanceTypeQuery endpoint.Endpoint
}

// MakeEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeEndpoints(s InstanceTypeService) Endpoints {
	return Endpoints{
		InstanceTypeQuery: kitoc.TraceEndpoint("cloudinfo.InstanceTypeQuery")(MakeInstanceTypeQueryEndpoint(s)),
	}
}

type instanceTypeQueryRequest struct {
	Provider string
	Service  string
	Region   *string
	Zone     *string
	Filter   graphql.InstanceTypeQueryInput
}

type instanceTypeQueryResponse struct {
	InstanceTypes []graphql.InstanceType
	Err           error
}

func (r instanceTypeQueryResponse) Failed() error {
	return r.Err
}

// MakeInstanceTypeQueryEndpoint returns an endpoint for the matching method of the underlying service.
func MakeInstanceTypeQueryEndpoint(s InstanceTypeService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(instanceTypeQueryRequest)

		query := cloudinfo.InstanceTypeQuery{
			Region: req.Region,
			Zone:   req.Zone,
			Filter: req.Filter,
		}

		instanceTypes, err := s.Query(ctx, req.Provider, req.Service, query)

		if err != nil {
			if _, ok := errors.Cause(err).(cloudinfo.InstanceTypeQueryValidationError); ok {
				return instanceTypeQueryResponse{
					Err: &instanceTypeError{
						msg:  err.Error(),
						code: codeInvalidInstanceTypeQuery,
					},
				}, nil
			}

			return nil, err
		}

		resp := instanceTypeQueryResponse{
			InstanceTypes: instanceTypes,
		}

		return resp, nil
	}
}
