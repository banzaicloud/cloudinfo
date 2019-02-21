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

package cloudinfo

import "context"

type eventBus interface {
	Publish(topic string, args ...interface{})
}

const loadConfig = "load_config"

type loaderEventBus struct {
	eb eventBus
}

func NewLoaderEvents(eb eventBus) *loaderEventBus {
	return &loaderEventBus{
		eb: eb,
	}
}

func (c *loaderEventBus) LoadConfig(ctx context.Context) {
	c.eb.Publish(loadConfig, ctx)
}
