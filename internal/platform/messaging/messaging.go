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

package messaging

import evbus "github.com/asaskevich/EventBus"

// EventBus event bus abstraction for the application to decouple vendor or lib specifics
type EventBus interface {
	Subscribe(topic string)
	Unsubscribe(topic string)
	Publish(topic string, args ...interface{})
}

type defaultEventBus struct {
	eventBus evbus.Bus
	fn       interface{}
}

func (eb *defaultEventBus) Subscribe(topic string) {
	eb.eventBus.SubscribeAsync(topic, eb.fn, false)
}

func (eb *defaultEventBus) Unsubscribe(topic string) {
	eb.eventBus.Unsubscribe(topic, nil)
}

func (eb *defaultEventBus) Publish(topic string, args ...interface{}) {
	eb.eventBus.Publish(topic, args)
}

//NewDefaultEventBus creates an event bus backed by  https://github.com/asaskevich/EventBus
func NewDefaultEventBus(callback interface{}) EventBus {

	return &defaultEventBus{
		eventBus: evbus.New(),
		fn:       callback,
	}

}
