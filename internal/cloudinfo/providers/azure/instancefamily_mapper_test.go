// Copyright © 2021 Banzai Cloud
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

package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"logur.dev/logur"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/cloudinfoadapter"
)

func TestAzureInfoer_mapSeries(t *testing.T) {
	familySeriesMap := map[string]string{
		"standardDADSv5Family": "DADSv5",
	}

	azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(&logur.TestLogger{})}

	for family, series := range familySeriesMap {
		t.Run("test parsing "+family, func(t *testing.T) {
			assert.Equal(t, azureInfoer.mapSeries(family), series, "unexpected series")
		})
	}
}
