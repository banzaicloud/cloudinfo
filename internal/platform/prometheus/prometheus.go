// Copyright © 2018 Banzai Cloud
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
package prometheus

import (
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.opencensus.io/exporter/prometheus"
)

// NewExporter creates a new, configured Prometheus exporter.
func NewExporter(config Config, errorHandler emperror.Handler) (*prometheus.Exporter, error) {
	exporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: config.Namespace,
		OnError:   errorHandler.Handle,
	})

	return exporter, errors.Wrap(err, "failed to create prometheus exporter")
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	return nil
}
