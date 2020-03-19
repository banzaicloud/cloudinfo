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
	"time"

	"github.com/go-kit/kit/endpoint"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

// LogEndpointDefaultName is the default endpoint name to use for logging.
const LogEndpointDefaultName = "gokit/endpoint"

// LogEndpoint returns an Endpoint middleware, logging a Go kit endpoint.
func LogEndpoint(name string, logger cloudinfo.Logger) endpoint.Middleware {
	if name == "" {
		name = LogEndpointDefaultName
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			logger := logger.WithContext(ctx)

			logger.Debug("processing request", map[string]interface{}{
				"endpoint": name,
			})

			defer func(begin time.Time) {
				logger.Debug("processing request finished", map[string]interface{}{
					"endpoint": name,
					"took":     time.Since(begin),
				})
			}(time.Now())

			response, err = next(ctx, request)
			return
		}
	}
}
