package api

import (
	"context"
	"errors"

	"github.com/goph/emperror"

	search "github.com/banzaicloud/cloudinfo/.gen/api/graphql"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct {
	cloudInfo cloudinfo.CloudInfo
}

func (r *Resolver) Query() search.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct {
	*Resolver
}

func (r *queryResolver) InstanceTypes(ctx context.Context, provider string, service *string, region *string, zone *string, filter search.InstanceTypeQueryInput) ([]search.InstanceType, error) {

	if provider == "" {
		return nil, errors.New("provider must be set")
	}

	var (
		raw  []cloudinfo.ProductDetails
		err  error
		data []search.InstanceType = make([]search.InstanceType, 0)
	)
	// load the data from the store
	// todo handle nil arguments (eg. retrieve multiple services and regions)
	if raw, err = r.cloudInfo.GetProductDetails(provider, *service, *region); err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve product details")
	}

	// filter the data
	for _, productDetails := range raw {

		var includeInResults = true

		if filter.Price != nil {
			includeInResults = includeInResults && applyFloatFilter(productDetails.OnDemandPrice, *filter.Price)
		}

		if filter.CPU != nil {
			includeInResults = includeInResults && applyFloatFilter(productDetails.Cpus, *filter.CPU)
		}

		if filter.Memory != nil {
			includeInResults = includeInResults && applyFloatFilter(productDetails.Mem, *filter.Memory)
		}

		if filter.NetworkCategory != nil {
			includeInResults = includeInResults && applyNetworkCategoryFilter(productDetails.NtwPerfCat, *filter.NetworkCategory)
		}

		if includeInResults {
			data = append(data, *r.transform(productDetails))
		}

	}

	// return the results

	return data, nil
}

func (r *queryResolver) transform(details cloudinfo.ProductDetails) *search.InstanceType {
	it := search.InstanceType{}

	it.Price = details.OnDemandPrice
	it.Name = details.Type
	it.CPU = details.Cpus
	it.Memory = details.Mem
	it.NetworkCategory = search.NetworkCategory(details.NtwPerfCat)

	return &it
}
