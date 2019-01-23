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

package logger

import (
	"context"
	"testing"

	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

func TestContextLogger(t *testing.T) {

	Init(logur.NewTestLogger())
	ctx := ToContext(context.Background(), NewLogCtxBuilder().
		WithProvider("test-provider").
		WithRegion("test-region").
		Build())
	Extract(ctx).Info("before routine")

	done := make(chan bool)

	go func(ctx context.Context, m chan bool) {
		ctx1 := ToContext(ctx, NewLogCtxBuilder().WithField("inroutine", "haha").Build())
		Extract(ctx1).Info("in routine")
		m <- true
	}(ctx, done)
	<-done
	Extract(ctx).Info("after routine")
}

func TestToContext(t *testing.T) {
	tests := []struct {
		name           string
		initialContext context.Context
		fields         map[string]interface{}
		check          func(lCtx context.Context, fields map[string]interface{})
	}{
		{
			name:           "no parent logging context",
			initialContext: context.Background(),
			fields:         NewLogCtxBuilder().WithProvider("prv1").WithService("srvc1").WithRegion("reg").Build(),
			check: func(lCtx context.Context, fields map[string]interface{}) {
				assert.Equal(t, lCtx.Value(ctxKey), fields)
			},
		},
		{
			name:           "parent logging context should be kept",
			initialContext: context.WithValue(context.TODO(), ctxKey, NewLogCtxBuilder().WithProvider("pr1").Build()),
			fields:         nil,
			check: func(lCtx context.Context, fields map[string]interface{}) {
				val := lCtx.Value(ctxKey).(map[string]interface{})
				assert.Equal(t, "pr1", val["provider"])
			},
		},
		{
			name:           "parent logging context value should be overwritten",
			initialContext: context.WithValue(context.Background(), ctxKey, NewLogCtxBuilder().WithProvider("pr1")),
			fields:         NewLogCtxBuilder().WithProvider("pr2").Build(),
			check: func(lCtx context.Context, fields map[string]interface{}) {
				val, _ := lCtx.Value(ctxKey).(map[string]interface{})
				assert.Equal(t, "pr2", val["provider"])
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := ToContext(test.initialContext, test.fields)
			test.check(ctx, test.fields)
		})
	}
}
