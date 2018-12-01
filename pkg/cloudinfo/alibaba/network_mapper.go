// Copyright © 2018 Banzai Cloud
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

package alibaba

import (
	"fmt"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

var (
	ntwPerfMap = map[string][]string{
		cloudinfo.NTW_LOW:    {"0.1 Gbit/s", "0.2 Gbit/s", "0.4 Gbit/s", "0.5 Gbit/s", "0.8 Gbit/s", "1.0 Gbit/s", "1.2 Gbit/s", "1.5 Gbit/s", "2.0 Gbit/s"},
		cloudinfo.NTW_MEDIUM: {"2.5 Gbit/s", "3.0 Gbit/s", "4.0 Gbit/s", "4.5 Gbit/s", "5.0 Gbit/s", "6.0 Gbit/s", "8.0 Gbit/s"},
		cloudinfo.NTW_HIGH:   {"10.0 Gbit/s"},
		cloudinfo.NTW_EXTRA:  {"17.0 Gbit/s", "25.0 Gbit/s"},
	}
)

// AlibabaNetworkMapper module object for handling Alibaba Cloud specific VM to Networking capabilities mapping
type AlibabaNetworkMapper struct {
}

// newAlibabaNetworkMapper initializes the network performance mapper struct
func newAlibabaNetworkMapper() *AlibabaNetworkMapper {
	return &AlibabaNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the alibaba instance to the category supported by telescopes
func (nm *AlibabaNetworkMapper) MapNetworkPerf(ntwPerf string) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if cloudinfo.Contains(strVals, ntwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", ntwPerf)
}
