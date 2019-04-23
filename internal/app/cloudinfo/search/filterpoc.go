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

package search

type Instance struct {
	cpu    int
	memory int
	disk   int
}

type IntFilter struct {
	lt  *int
	eq  *int
	gt  *int
	ne  *int
	in  []int
	nin [] int
}

type InstanceQueryInput struct {
	cpu    Filter
	memory Filter
	disk   Filter
}

type Query interface {
	instances(filter InstanceQueryInput) []Instance
}

// Filter is a marker interface for filters
type Filter interface {
	applies(instance interface{}) bool
}

// Evaluator evaluates an instance against the filter
type Evaluator interface {
	evaluate(instance interface{}, filter Filter)
}

// InputEvaluator component in charge to process filters passed in in the input
type InputEvaluator struct {

}

// QueryExecutor component struct for executing a query
type QueryExecutor struct {
	iputEvaluator Evaluator
}

func (qe *QueryExecutor) instances(filter InstanceQueryInput) []Instance {
	// retrieve all the data from the store
	// iterate over the data
	// apply the relevant filters
	return nil
}
