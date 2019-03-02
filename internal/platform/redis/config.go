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

package redis

import (
	"fmt"

	"github.com/pkg/errors"
)

// Config holds information necessary for connecting to Redis.
type Config struct {
	// Host is the Redis host.
	Host string

	// Port is the Redis port.
	Port int

	// Password list supports passing multiple passwords making password changes easier
	Password []string
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("redis host is required")
	}

	if c.Port == 0 {
		return errors.New("redis port is required")
	}

	return nil
}

// Server returns the host-port pair for the connection.
func (c Config) Server() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
