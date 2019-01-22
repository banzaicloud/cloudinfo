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

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/problems"
	"github.com/gin-gonic/gin"
)

// Responder marks responders
type Responder interface {
	// Respond implements the responding logic / it's intended to be self-contained
	Respond(ginCtx *gin.Context, err error)
}

// errorResponder struct in charge for assembling classified error responses
type errorResponder struct {
	errClassifier Classifier
}

// Respond assembles the error response corresponding to the passed in error
func (er *errorResponder) Respond(ginCtx *gin.Context, err error) {

	if responseData, e := er.errClassifier.Classify(err); e == nil {
		er.respond(ginCtx, responseData)
		return
	}

	er.respond(ginCtx, problems.NewUnknownProblem(err))
}

// respond sets the response in the gin context
func (er *errorResponder) respond(ginCtx *gin.Context, d interface{}) {

	if problems.IsDefaultProblem(d) {
		ginCtx.JSON(problems.ProblemStatus(d), d)
		ginCtx.Abort()
		return
	}

	ginCtx.JSON(http.StatusInternalServerError, problems.NewUnknownProblem(d))
}

// NewErrorResponder configures a new error responder
func NewErrorResponder() Responder {
	return &errorResponder{
		errClassifier: NewErrorClassifier(),
	}
}
