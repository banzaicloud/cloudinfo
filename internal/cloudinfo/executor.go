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
	"time"
)

// TaskFn function type for executing task logic
type TaskFn func(c context.Context)

// Executor
type Executor interface {
	Execute(ctx context.Context, sf TaskFn) error
}

// PeriodicExecutor Executor that periodically executes the passed in task function
type PeriodicExecutor struct {
	// interval specifies the time interval within the task function will be executed once
	interval time.Duration
	log      Logger
}

// Execute executes the task function periodically in a new goroutine
// For tasks that need to be periodically executed within a defined deadline, the appropriate context needs to be passed in
func (ps *PeriodicExecutor) Execute(ctx context.Context, sf TaskFn) error {
	go sf(ctx)

	ticker := time.NewTicker(ps.interval)
	go func(c context.Context) {
		for {
			select {
			case <-ticker.C:
				sf(c)
			case <-c.Done():
				ps.log.Debug("stopping periodic execution")
				ticker.Stop()
				return
			}
		}
	}(ctx)

	return nil
}

// NewPeriodicExecutor creates a new Executor with the given time period
func NewPeriodicExecutor(period time.Duration, log Logger) Executor {
	return &PeriodicExecutor{
		interval: period,
		log:      log,
	}
}
