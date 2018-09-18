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
)

func Test(t *testing.T) {

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
