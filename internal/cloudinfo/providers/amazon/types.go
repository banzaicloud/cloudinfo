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
	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
)

// PricingSource list of operations for retrieving pricing information
// Decouples the pricing logic from the amazon api
type PricingSource interface {
	GetPriceList(input *pricing.GetProductsInput) ([]aws.JSONValue, error)
}

// pricingDetails wraps a pricing client, and implements the PricingSource interface
type pricingDetails struct {
	// embedded pricing client
	pricing.Pricing
}

func NewPricingSource(s *session.Session, cfg ...*aws.Config) *pricingDetails {
	return &pricingDetails{
		*pricing.New(s, cfg...),
	}
}

func (pd *pricingDetails) GetPriceList(input *pricing.GetProductsInput) ([]aws.JSONValue, error) {
	list := make([]aws.JSONValue, 0)

	if err := pd.GetProductsPages(input, func(output *pricing.GetProductsOutput, b bool) bool {
		list = append(list, output.PriceList...)
		return !b
	}); err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve pricelist")
	}

	return list, nil
}
