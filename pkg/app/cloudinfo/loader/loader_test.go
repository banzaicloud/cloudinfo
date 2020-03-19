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

package loader

import (
	"testing"
)

func TestDefaultServiceLoader_Load(t *testing.T) {

	// config := Config{
	// 	SvcDataLocation:        ".",
	// 	SvcDefinitionsLocation: ".",
	// 	Name:                   "service-definition",
	// }
	// l := logrus.New()
	// level, _ := logrus.ParseLevel("debug")
	// l.SetLevel(level)
	//
	// log := logrusadapter.New(l)
	//
	// store := cloudinfo.NewCacheProductStore(10*time.Minute, 10*time.Minute, log)
	//
	// loader := NewDefaultServiceLoader(config, store, log)
	//
	// //loader.LoadServiceInformation(context.Background())
	// loader.ConfigureServices(context.Background())
	//
	// reg, _ := store.GetRegions("test-prv", "test-svc")
	// log.Info("stored", map[string]interface{}{"cnt": reg})

}
