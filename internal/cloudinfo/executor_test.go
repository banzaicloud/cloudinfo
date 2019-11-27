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

import (
	"context"
	"testing"
	"time"
)

func TestPeriodicExecutor_Execute(t *testing.T) {
	tests := []struct {
		name   string
		period time.Duration
		ctx    func() (context.Context, context.CancelFunc)
		task   TaskFn

		checker func(context.Context, context.CancelFunc)
	}{
		{
			name:   "execute periodically till deadline exceeded",
			period: 1 * time.Second,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
			},
			task: func(c context.Context) {
				println("do nothing ...")
				c.Done()
			},

			checker: func(c context.Context, cFn context.CancelFunc) {
				// wait till done
				<-c.Done()
			},
		},
		{
			name:   "execute periodically no deadline",
			period: 1 * time.Second,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},

			task: func(c context.Context) {
				println("do nothing ...")
				c.Done()
			},
			checker: func(c context.Context, cancelFunc context.CancelFunc) {
				// cancel after 5 sec
				time.Sleep(5 * time.Second)
				cancelFunc()
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cFn := test.ctx()
			if err := NewPeriodicExecutor(test.period, cloudinfoLogger).Execute(ctx, test.task); err != nil {
				t.Fail()
			}
			test.checker(ctx, cFn)
		})
	}
}
