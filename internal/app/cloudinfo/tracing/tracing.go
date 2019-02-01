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
	// StartSpan starts a span with the given name and context
	StartSpan(ctx context.Context, name string) (context.Context, *CiSpan)

	// StartWithTags starts a new span and adds the tags to it as attributes
	StartWithTags(ctx context.Context, name string, tags map[string]interface{}) (context.Context, *CiSpan)

	// StartAndLink starts a new root span and links it with the span from the passed in context
	StartAndLink(parentCtx context.Context, name string) (context.Context, *CiSpan)

	// EndSpan ends a span in the given context
	EndSpan(ctx context.Context)

	EndSpanInstance(span *CiSpan)
}

type CiSpan struct {
	*trace.Span
}

type ciTracer struct {
}

// StartAndLink starts a new root span and links it to the span in the provided context if any
func (t *ciTracer) StartAndLink(parentCtx context.Context, name string) (context.Context, *CiSpan) {
	// start a new root span
	newCtx, rootSpan := trace.StartSpan(context.Background(), name)

	// get the spant to link from the context if any
	if linkSpan := trace.FromContext(parentCtx); linkSpan != nil {
		rootSpan.AddLink(trace.Link{
			SpanID:  linkSpan.SpanContext().SpanID,
			TraceID: linkSpan.SpanContext().TraceID,
			Type:    trace.LinkTypeChild,
		})
	}

	return newCtx, &CiSpan{rootSpan}

}

func (t *ciTracer) EndSpanInstance(span *CiSpan) {
	span.End()
}

func (t *ciTracer) StartWithTags(ctx context.Context, name string, tags map[string]interface{}) (context.Context, *CiSpan) {
	var attrs []trace.Attribute
	ctx, span := t.StartSpan(ctx, name)

	for k, v := range tags {
		switch v.(type) {
		case string:
			attrs = append(attrs, trace.StringAttribute(k, v.(string)))
		case bool:
			attrs = append(attrs, trace.BoolAttribute(k, v.(bool)))
		case int64:
			attrs = append(attrs, trace.Int64Attribute(k, v.(int64)))
		}
	}
	if len(attrs) > 0 {
		span.AddAttributes(attrs...)
	}
	return ctx, span
}

// EndSpan ends the span in the given context
func (t *ciTracer) EndSpan(ctx context.Context) {
	var span *trace.Span
	if span = trace.FromContext(ctx); span == nil {
		// there's no span in the context
		return
	}
	span.End()
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
