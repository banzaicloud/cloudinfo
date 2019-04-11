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

import (
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/sirupsen/logrus"
)

type query struct {
	// supported set of operators
	operators *operators

	log logur.Logger
}

func (q *query) isOperator(mapKey string) (operator, bool) {
	op, ok := q.operators.opMap[mapKey]
	return op, ok
}

func (q *query) operatorKind(mapKey string) string {
	o, _ := q.operators.opMap[mapKey]
	return o.kind
}

// walk walks the model and creates the filterchain
func (q *query) walk(queryData interface{}, fc filterChain) filterChain {

	var (
		m  (map[string]interface{})
		ok bool
	)

	// data must be a map
	if m, ok = queryData.(map[string]interface{}); !ok {
		q.log.Error("query model not a map of the existing format")
		return nil
	}

	for k, v := range m {
		var (
			op operator
			ok bool
		)

		op, ok = q.isOperator(k)

		if !ok {
			//a "field" reached, the value should be a "leaf" (map keyed with the operator name)
			q.log.Info("processing field", map[string]interface{}{"field": k})

			// create filters for the field
			fc.chain(q.getFilters(k, v)...)

			continue
		}

		q.log.Info("processing operator", map[string]interface{}{"op": k})

		// the operator is a "logical one"
		switch opKind := op.name; opKind {
		case "$or":

			q.log.Info("creating new chain", map[string]interface{}{"kind": opKind})
			if len(fc.filterList()) > 0 {
				fc.chain(OrChain())
			} else {
				fc = OrChain()
			}
		case "$and":
			q.log.Info("creating new chain", map[string]interface{}{"kind": opKind})
			if len(fc.filterList()) > 0 {
				fc.chain(AndChain())
			} else {
				fc = AndChain()
			}

		}

		// recursive call - process the subtree for a logical operator
		fc = q.walk(v, fc)

	}

	return fc

}

// getFilters assembles filters for the given fields
func (q *query) getFilters(field string, opToVal interface{}) []filter {

	var (
		filters = make([]filter, 0)
		leaf    map[string]interface{}
		ok      bool
	)

	// data should always be a map
	if leaf, ok = opToVal.(map[string]interface{}); !ok {
		return nil
	}

	for op, val := range leaf {
		opr := q.operators.opMap[op]
		filters = append(filters, NewFilter(field, opr, val))
	}

	q.log.Info("created filters", map[string]interface{}{"count": len(filters)})
	return filters
}

func NewQuery(ops *operators, model queryModel) *query {
	q := query{
		operators: ops,
		log:       logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"comp": "walk"}),
	}
	return &q
}

// filter defines operations for checking a record against the (transformed) walk
type filter interface {
	// applies operation for checkong the filter criteria on the passed in instance
	applies(instance interface{}) bool
}

// chained filters
type filterChain interface {
	filter

	// adds a filter to the chain
	chain(filter ...filter)

	filterList() []filter
}
type logicalFilter struct {
	kind    string // and / or
	filters []filter
}

func (lf *logicalFilter) applies(instance interface{}) bool {
	var applies = true
	for _, filter := range lf.filters {

		applies = filter.applies(instance)

		switch kind := lf.kind; kind {
		case "AND":
			if !applies {
				// return on the first filter that doesn't apply
				return false
			}
		case "OR":
			if applies {
				// return on the first filter that applies
				return true
			}
		}

	}
	return applies
}

// implement the chain interface
func (lf *logicalFilter) chain(f ...filter) {

	if lf.kind == "" {
		// the default chain is AND
		lf.kind = "AND"
	}

	lf.add(f...)
}

func (lf *logicalFilter) filterList() []filter {
	return lf.filters
}

func (lf *logicalFilter) add(fltr ...filter) {
	if lf.filters == nil {
		lf.filters = make([]filter, 0, 0)
	}

	lf.filters = append(lf.filters, fltr...)
}

//
func AndChain(filters ...filter) filterChain {
	if filters == nil {
		filters = make([]filter, 0, 0)
	}
	lf := logicalFilter{
		kind:    "AND",
		filters: filters,
	}

	return &lf
}

func OrChain(filters ...filter) filterChain {

	if filters == nil {
		filters = make([]filter, 0, 0)
	}

	lf := logicalFilter{
		kind:    "OR",
		filters: filters,
	}

	return &lf
}

// operatorFilter "generic" filter in charge for applying an operator
type operatorFilter struct {
	// the name of the struct to be checked
	field string

	// the value to be compared
	value interface{}

	// the
	operator operator
}

func (of *operatorFilter) applies(instance interface{}) bool {
	return of.operator.opFunc(instance, of.field, of.value)
}

func NewFilter(field string, op operator, value interface{}) filter {
	f := operatorFilter{
		field:    field,
		operator: op,
		value:    value,
	}
	return &f
}
