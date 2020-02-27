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

// Package log configures a new logger for an application.
package log

import (
	"fmt"
	"os"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/sirupsen/logrus"
)

// NewLogger creates a new logger.
func NewLogger(config Config) logur.Logger {
	logger := logrus.New()

	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:             config.NoColor,
		EnvironmentOverrideColors: true,
	})

	switch config.Format {
	case "logfmt":
		// Already the default

	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	if level, err := logrus.ParseLevel(config.Level); err == nil {
		logger.SetLevel(level)
	}

	return logrusadapter.New(logger)
}

// WithFields returns a new contextual logger instance with context added to it.
func WithFields(logger logur.Logger, fields map[string]interface{}) logur.Logger {
	return logur.WithFields(logger, fields)
}

const correlationIdField = "correlation-id"

// WithFieldsForHandlers returns a new logger instance with a correlation ID in it.
func WithFieldsForHandlers(ctx *gin.Context, logger logur.Logger, fields map[string]interface{}) logur.Logger {
	cid := ctx.GetString(ContextKey)

	if cid == "" {
		return logur.WithFields(logger, fields)
	}

	if fields == nil {
		fields = make(map[string]interface{}, 1)
	}

	fields[correlationIdField] = cid

	return logur.WithFields(logger, fields)
}

// ToMap creates a map of key-value pairs.
//
// The implementation bellow is from go-kit's JSON logger.
func ToMap(keyvals []interface{}) map[string]interface{} {
	m := map[string]interface{}{}

	if len(keyvals) == 0 {
		return m
	}

	if len(keyvals)%2 == 1 {
		keyvals = append(keyvals, nil)
	}

	for i := 0; i < len(keyvals); i += 2 {
		merge(m, keyvals[i], keyvals[i+1])
	}

	return m
}

func merge(dst map[string]interface{}, k, v interface{}) {
	var key string

	switch x := k.(type) {
	case string:
		key = x
	case fmt.Stringer:
		key = safeString(x)
	default:
		key = fmt.Sprint(x)
	}

	switch x := v.(type) {
	case error:
		v = safeError(x)
	case fmt.Stringer:
		v = safeString(x)
	}

	dst[key] = v
}

func safeString(str fmt.Stringer) (s string) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if v := reflect.ValueOf(str); v.Kind() == reflect.Ptr && v.IsNil() {
				s = "NULL"
			} else {
				panic(panicVal)
			}
		}
	}()

	s = str.String()

	return
}

func safeError(err error) (s interface{}) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if v := reflect.ValueOf(err); v.Kind() == reflect.Ptr && v.IsNil() {
				s = nil
			} else {
				panic(panicVal)
			}
		}
	}()

	s = err.Error()

	return
}
