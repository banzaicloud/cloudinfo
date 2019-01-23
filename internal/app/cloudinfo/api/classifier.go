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

package api

import (
	"net/http"
	"net/url"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/problems"
	"github.com/go-openapi/runtime"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// Classifier represents a contract to classify passed in structs
type Classifier interface {
	// Classify classifies the passed in struct based on arbitrary, implementation specific criteria
	Classify(in interface{}) (interface{}, error)
}

// errClassifier type implementing the Classifier interface
type errClassifier struct {
}

// NewErrorClassifier returns a reference to a classifier instance
func NewErrorClassifier() Classifier {
	return &errClassifier{}
}

// Classify classification implementation based on the error and its context
func (erc *errClassifier) Classify(inErr interface{}) (interface{}, error) {
	var (
		err     error
		ok      bool
		problem *problems.ProblemWrapper
	)

	if err, ok = inErr.(error); !ok {
		return nil, errors.New("failed to classify error")
	}

	cause := errors.Cause(err)

	switch e := cause.(type) {

	case *runtime.APIError:
		// (cloud info) service is reachable - operation failed (eg.: bad request)
		problem = erc.classifyApiError(e, emperror.Context(err))
	case *url.Error:
		// the cloud info service is not available
		problem = erc.classifyUrlError(e, emperror.Context(err))
	default:
		// unclassified error
		problem = erc.classifyGenericError(cause, emperror.Context(err))
	}

	return problem, nil

}

// classifyApiError assembles data to be sent in the response to the caller when the error originates from the cloud info service
func (erc *errClassifier) classifyApiError(e *runtime.APIError, ctx []interface{}) *problems.ProblemWrapper {

	var (
		httpCode int
		details  = "unknown failure"
	)

	// determine http status code
	switch c := e.Code; {
	case c < http.StatusInternalServerError:
		// all non-server error status codes translated to user error status code
		httpCode = http.StatusBadRequest
	default:
		// all server errors left unchanged
		httpCode = c
	}

	// determine error code and status message - from the error and the context
	// the message should contain the flow related information and
	if hasLabel(ctx, "validation") {
		// provider, service, region - path data
		details = "validation failed - no cloud information available for the request path data"
		return problems.NewValidationProblem(httpCode, details)
	}

	return problems.NewDetailedProblem(httpCode, details)
}

func (erc *errClassifier) classifyUrlError(e *url.Error, ctx []interface{}) *problems.ProblemWrapper {
	// todo
	var problem = problems.NewUnknownProblem(e)

	return problem
}

func (erc *errClassifier) classifyGenericError(e error, ctx []interface{}) *problems.ProblemWrapper {
	// todo
	var problem = problems.NewUnknownProblem(e)

	if hasLabel(ctx, "validation") {
		problem = problems.NewValidationProblem(http.StatusBadRequest, e.Error())
	}

	return problem
}

func hasLabel(ctx []interface{}, s interface{}) bool {
	for _, e := range ctx {
		if e == s {
			return true
		}
	}
	return false
}
