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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"reflect"
	"testing"
)

func TestParseQuery(t *testing.T) {

	op := NewOperators()
	//raw := `{"$or":{"_changed":{"$gt":{"$date":"2016-08-01"},"$lt":{"$date":"2016-08-05"}}}}`
	//raw := `{"_changed":{"$gt":{"$date":"2016-08-01"},"$lt":{"$date":"2016-08-05"}}}`
	raw := `{"$and":{"Cpus":{"$gt":1, "$lt": 5}}}`

	qm := queryModel{}
	qm.parse(raw)

	qr := NewQuery(op, qm)
	f := qr.walk(qm.m, &logicalFilter{})

	vm := cloudinfo.VmInfo{
		Cpus: 0,
		Mem:  64,
	}

	f.applies(vm)

	fmt.Println(f)

}

type testX struct {
	fld string
	nbr string
}

func TestRefl(t *testing.T) {

	var x interface{}

	x = testX{fld: "field", nbr: "2"}

	v := reflect.ValueOf(x)

	fmt.Println(v.FieldByName("nbr"))

}
