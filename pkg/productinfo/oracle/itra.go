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

package oracle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/banzaicloud/productinfo/pkg/logger"
)

// ITRAProductInfo holds information of a product
type ITRAProductInfo struct {
	PartNumber string          `json:"partNumber"`
	Prices     []ITRAPriceInfo `json:"prices"`
}

// ITRAPriceInfo holds price information
type ITRAPriceInfo struct {
	Model string  `json:"model"`
	Value float64 `json:"value"`
}

// ITRAResponse represents a response of ITRA product info request
type ITRAResponse struct {
	Items   []ITRAProductInfo `json:"items"`
	Link    string            `json:"canonicalLink"`
	HasMode bool              `json:"hasMore"`
	Limit   uint              `json:"limit"`
	Offset  uint              `json:"offset"`
}

// GetProductInfoFromITRA gets product information from ITRA api by part number
func (i *Infoer) GetProductInfoFromITRA(ctx context.Context, partNumber string) (info ITRAProductInfo, err error) {

	if i.productInfoCache == nil {
		i.productInfoCache = make(map[string]ITRAProductInfo)
	}

	if _, ok := i.productInfoCache[partNumber]; ok {
		logger.Extract(ctx).Debugf("getting product info for PN[%s] - from cache", partNumber)
		return i.productInfoCache[partNumber], nil
	}

	logger.Extract(ctx).Debugf("getting product info for PN[%s]", partNumber)

	url := fmt.Sprintf("https://itra.oraclecloud.com/itas/.anon/myservices/api/v1/products?partNumber=%s", partNumber)
	resp, err := http.Get(url)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	var response ITRAResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	respByte := buf.Bytes()
	if err := json.Unmarshal(respByte, &response); err != nil {
		return info, err
	}

	if len(response.Items) < 1 {
		return info, fmt.Errorf("No product information was found for PN[%s]", partNumber)
	}

	i.productInfoCache[partNumber] = response.Items[0]
	return i.productInfoCache[partNumber], nil
}

// GetPrice gets the value of the given price model from gathered prices
func (ipi *ITRAProductInfo) GetPrice(model string) float64 {

	for _, price := range ipi.Prices {
		if model == price.Model {
			return price.Value
		}
	}

	return 0
}
