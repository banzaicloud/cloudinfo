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

import (
	"strings"

	"emperror.dev/emperror"
	evbus "github.com/asaskevich/EventBus"
)

// EventBus event bus abstraction for the application to decouple vendor or lib specifics

type EventBus interface {
	// PublishScrapingComplete emits a "scraping complete" message for the given provider
	PublishScrapingComplete(provider string)

	// SubscribeScrapingComplete
	SubscribeScrapingComplete(provider string, callback interface{})
}

const (
	topicPrefix = "load:service"
)

// defaultEventBus default EventBus component implementation backed by https://github.com/asaskevich/EventBus
type defaultEventBus struct {
	eventBus     evbus.Bus
	errorHandler emperror.ErrorHandler
}

func (eb *defaultEventBus) PublishScrapingComplete(provider string) {
	eb.eventBus.Publish(eb.providerScrapingTopic(provider))
}

func (eb *defaultEventBus) SubscribeScrapingComplete(provider string, callback interface{}) {
	if err := eb.eventBus.SubscribeAsync(eb.providerScrapingTopic(provider), callback, false); err != nil {
		eb.errorHandler.Handle(err)
	}
}

func (eb *defaultEventBus) providerScrapingTopic(provider string) string {
	return strings.Join([]string{topicPrefix, provider}, ":")
}

// NewDefaultEventBus creates an event bus backed by  https://github.com/asaskevich/EventBus
func NewDefaultEventBus(_ emperror.ErrorHandler) EventBus {
	return &defaultEventBus{
		eventBus: evbus.New(),
	}
}
