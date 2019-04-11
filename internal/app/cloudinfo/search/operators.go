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
	"fmt"
	"reflect"
)

const (
	conditional = "conditional"
	logical     = "logical"
)

type operators struct {
	// opname -> op
	opMap map[string]operator
}

// operator represents a supported operation that can be applied to a data "record"
type operator struct {
	name   string
	kind   string
	opFunc func(instance interface{}, field string, value interface{}) bool
}

func (o *operator) String() string {
	return fmt.Sprintf("%#v", o)
}

// ltOperator lower than operator
func ltOperator() operator {
	o := operator{
		kind: conditional,
		name: "$lt",
		opFunc: func(instance interface{}, field string, value interface{}) bool {

			// todo muhaha
			// get the value of the struct (as string!!!)
			i := reflect.ValueOf(instance)
			fieldVal := fmt.Sprintf("%s", i.FieldByName(field))
			cond := fmt.Sprintf("%s", value)

			return fieldVal < cond
		},
	}
	return o
}

// gtOperator greather than operator
func gtOperator() operator {
	o := operator{
		kind: conditional,
		name: "$gt",
		opFunc: func(instance interface{}, field string, value interface{}) bool {

			// todo muhaha
			// get the value of the struct (as string!!!)
			i := reflect.ValueOf(instance)
			fieldVal := fmt.Sprintf("%s", i.FieldByName(field))
			cond := fmt.Sprintf("%s", value)

			return fieldVal > cond
		},
	}
	return o
}

func (ops *operators) setup() {
	ops.opMap = make(map[string]operator)
	ops.opMap["$lt"] = ltOperator()
	ops.opMap["$gt"] = gtOperator()
	ops.opMap["$and"] = operator{name: "$and", kind: logical}
	ops.opMap["$or"] = operator{name: "$or", kind: logical}
}

func NewOperators() *operators {
	op := operators{}
	op.setup()
	return &op
}
