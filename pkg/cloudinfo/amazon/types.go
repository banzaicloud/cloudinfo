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

package amazon

import (
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
)

// PricingSource list of operations for retrieving pricing information
// Decouples the pricing logic from the amazon api
type PricingSource interface {
	GetAttributeValues(input *pricing.GetAttributeValuesInput) (*pricing.GetAttributeValuesOutput, error)
	GetPriceList(input *pricing.GetProductsInput) ([]aws.JSONValue, error)
}

// pricingDetails wraps a pricing client, and implements the PricingSource interface
type pricingDetails struct {
	// embedded pricing client
	pricing.Pricing

	// temporary price list, this only is to workaround the pagination results
	tmpPl []aws.JSONValue

	// used for locking access to the tmpPl during paginated retrievals
	mtx sync.Mutex
}

func NewPricingSource(s *session.Session, cfg *aws.Config) *pricingDetails {
	return &pricingDetails{
		*pricing.New(s, cfg),
		nil,
		sync.Mutex{},
	}
}

func (pd *pricingDetails) GetPriceList(input *pricing.GetProductsInput) ([]aws.JSONValue, error) {

	pd.mtx.Lock()
	defer pd.mtx.Unlock()
	// clear the cached pricelist
	pd.tmpPl = make([]aws.JSONValue, 0)

	if err := pd.GetProductsPages(input, pd.collectorFn()); err != nil {
		// todo use emperror and wrap the original error
		return nil, errors.New("failed to retrieve pricelist")
	}

	// transfer values to the temp list
	list := pd.tmpPl

	// clear the cached pricelist
	pd.tmpPl = nil

	return list, nil
}

func (pd *pricingDetails) collectorFn() func(output *pricing.GetProductsOutput, b bool) bool {

	return func(output *pricing.GetProductsOutput, b bool) bool {
		pd.tmpPl = append(pd.tmpPl, output.PriceList...)
		return !b
	}
}
