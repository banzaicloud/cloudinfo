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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestContextLogger(t *testing.T) {

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

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		fmtStr    string
		formatter interface{}
		checker   func(args ...interface{}) bool
	}{
		{
			name:     "debug log level",
			logLevel: "debug",
			fmtStr:   "",
			checker: func(logArg ...interface{}) bool {
				level := logArg[1]
				assert.Equal(t, level, Level())
				return true
			},
		},
		{
			name:     "info log level",
			logLevel: "info",
			fmtStr:   "",
			checker: func(logArg ...interface{}) bool {
				level := logArg[1]
				assert.Equal(t, level, Level())
				return true
			},
		},
		{
			name:     "default log formatter",
			logLevel: "error",
			fmtStr:   "",
			checker: func(logArg ...interface{}) bool {
				assert.Equal(t, reflect.TypeOf(&logrus.TextFormatter{}).String(), reflect.TypeOf(Formatter()).String())
				return true
			},
		},
		{
			name:     "json log formatter",
			logLevel: "error",
			fmtStr:   "json",
			checker: func(logArg ...interface{}) bool {
				assert.Equal(t, reflect.TypeOf(&logrus.JSONFormatter{}).String(), reflect.TypeOf(Formatter()).String())
				return true
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			InitLogger(test.logLevel, test.fmtStr)
			test.checker(Log(), test.logLevel, test.fmtStr)
		})
	}
}
