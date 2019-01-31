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

package tracing

import (
	"context"
	"github.com/banzaicloud/cloudinfo/internal/platform/jaeger"
	"github.com/goph/emperror"
	"go.opencensus.io/trace"
)

// Tracer represents the application specific view of tracing
// It's meant to collect all tracing related operations
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, *CiSpan)
}

type CiSpan struct {
	*trace.Span
}

type ciTracer struct {
}

func (t *ciTracer) StartSpan(ctx context.Context, name string) (context.Context, *CiSpan) {
	c, s := trace.StartSpan(ctx, name)
	return c, &CiSpan{s}
}

func NewTracer() Tracer {
	return &ciTracer{}
}

func SetupTracing(config jaeger.Config, errorHandler emperror.Handler) {
	exporter, err := jaeger.NewExporter(config, errorHandler)
	errorHandler.Handle(err)

	trace.RegisterExporter(exporter)

	// 2. Configure 100% sample rate, otherwise, few traces will be sampled.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}
