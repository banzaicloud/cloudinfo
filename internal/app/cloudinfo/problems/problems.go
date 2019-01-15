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
	"fmt"
	"github.com/moogar0880/problems"
	"net/http"
)

const (
	validationProblemTitle = "validation problem"
	providerProblemTitle   = "cloud provider problem"
)

// NewValidationProblem identif
func NewValidationProblem(code int, details string) *problems.DefaultProblem {
	pb := problems.NewDetailedProblem(code, details)
	pb.Title = validationProblemTitle
	return pb
}

func NewProviderProblem(code int, details string) *problems.DefaultProblem {
	pb := problems.NewDetailedProblem(code, details)
	pb.Title = providerProblemTitle
	return pb
}

func NewUnknownProblem(un interface{}) *problems.DefaultProblem {
	return problems.NewDetailedProblem(http.StatusInternalServerError, fmt.Sprintf("error: %#v", un))
}
