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
)

// for testing purposess
type noOpLogger struct {
}

func (d noOpLogger) Trace(msg string, fields ...map[string]interface{}) {
}

func (d noOpLogger) Debug(msg string, fields ...map[string]interface{}) {
}

func (d noOpLogger) Info(msg string, fields ...map[string]interface{}) {
}

func (d noOpLogger) Warn(msg string, fields ...map[string]interface{}) {
}

func (d noOpLogger) Error(msg string, fields ...map[string]interface{}) {
}

func (d noOpLogger) WithFields(fields map[string]interface{}) Logger {
	return d
}

func (d noOpLogger) WithContext(ctx context.Context) Logger {
	return d
}

func NoOpLogger() Logger {
	return noOpLogger{}
}
