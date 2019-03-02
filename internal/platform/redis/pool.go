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
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

type PoolWrapper struct {
	*redis.Pool
}

// NewPool creates a new redis connection pool.
func NewPool(config Config) *redis.Pool {
	return &redis.Pool{
		MaxIdle: 10,
		Wait:    true, // Wait for the connection pool, no connection pool exhausted error
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(
				"tcp",
				config.Server(),
			)
			if err != nil {
				return nil, errors.Wrap(err, "failed to dial redis server")
			}

			if len(config.Password) > 0 {
				var err error

				for _, password := range config.Password {
					_, err = c.Do("AUTH", password)
					if err == nil {
						break
					}
				}

				if err != nil {
					c.Close()

					return nil, errors.Wrap(err, "none of the provided passwords were accepted by the server")
				}
			}

			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")

			return err
		},
	}
}
