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

import "emperror.dev/errors"

// Config holds information necessary for sending trace to Jaeger.
type Config struct {
	// CollectorEndpoint is the Jaeger HTTP Thrift endpoint.
	// For example, http://localhost:14268.
	CollectorEndpoint string

	// AgentEndpoint instructs exporter to send spans to Jaeger agent at this address.
	// For example, http://localhost:14268/api/traces?format=jaeger.thrift.
	AgentEndpoint string

	// Username to be used if basic auth is required.
	// Optional.
	Username string

	// Password to be used if basic auth is required.
	// Optional.
	Password string

	// ServiceName is the name of the process.
	ServiceName string
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.CollectorEndpoint == "" && c.AgentEndpoint == "" {
		return errors.New("either collector endpoint or agent endpoint must be configured")
	}

	return nil
}
