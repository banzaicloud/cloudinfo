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

package problems

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/moogar0880/problems"
)

const (
	validationProblemTitle = "validation problem"
	providerProblemTitle   = "cloud provider problem"
)

type ProblemWrapper struct {
	*problems.DefaultProblem
}

func (pw *ProblemWrapper) String() string {
	str, _ := json.Marshal(pw.DefaultProblem)
	return string(str)
}

func NewValidationProblem(code int, details string) *ProblemWrapper {
	pb := problems.NewDetailedProblem(code, details)
	pb.Title = validationProblemTitle
	return &ProblemWrapper{pb}
}

func NewProviderProblem(code int, details string) *ProblemWrapper {
	pb := problems.NewDetailedProblem(code, details)
	pb.Title = providerProblemTitle
	return &ProblemWrapper{pb}
}

func NewUnknownProblem(un interface{}) *ProblemWrapper {
	return &ProblemWrapper{problems.NewDetailedProblem(http.StatusInternalServerError, fmt.Sprintf("%s", un))}
}

func IsDefaultProblem(d interface{}) bool {
	_, ok := d.(*ProblemWrapper)
	return ok
}

func NewDetailedProblem(status int, details string) *ProblemWrapper {
	return &ProblemWrapper{problems.NewDetailedProblem(status, details)}
}

func ProblemStatus(d interface{}) int {
	if pb, ok := d.(*ProblemWrapper); ok {
		return pb.Status
	}
	return -1
}
