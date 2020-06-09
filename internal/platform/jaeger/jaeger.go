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

package jaeger

import (
	"contrib.go.opencensus.io/exporter/jaeger"
	"emperror.dev/emperror"
	"emperror.dev/errors"
)

// NewExporter creates a new, configured Jaeger exporter.
func NewExporter(config Config, errorHandler emperror.ErrorHandler) (*jaeger.Exporter, error) {
	exporter, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: config.CollectorEndpoint,
		AgentEndpoint:     config.AgentEndpoint,
		Username:          config.Username,
		Password:          config.Password,
		OnError: emperror.HandlerWith(
			errorHandler,
			"component", "opencensus",
			"exporter", "jaeger",
		).Handle,
		Process: jaeger.Process{
			ServiceName: config.ServiceName,
		},
	})

	return exporter, errors.Wrap(err, "failed to create jaeger exporter")
}
