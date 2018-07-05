
[![CircleCI](https://circleci.com/gh/banzaicloud/productinfo/tree/master.svg?style=shield)](https://circleci.com/gh/banzaicloud/productinfo/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/productinfo)](https://goreportcard.com/report/github.com/banzaicloud/productinfo)
![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1651/badge)](https://bestpractices.coreinfrastructure.org/projects/1651)


# Product Info

The Banzai Cloud 'productinfo' application is a standalone project in the [Pipeline](https://github.com/banzaicloud/pipeline) ecosystem.
It's main purpose is to gather product information from the supported cloud providers and make them available to clients in a unified format

## Quick start

Building the project is as simple as running a go build command. The result is a statically linked executable binary.

```
go build .
```

The following options can be configured when starting the exporter (with defaults):

```
./productinfo --help
Usage of ./productinfo:
      --azure-subscription-id string             Azure subscription ID to use with the APIs
      --gce-api-key string                       GCE API key to use for getting SKUs
      --gce-project-id string                    GCE project ID to use
      --help                                     print usage
      --listen-address string                    the address the telescope listens to HTTP requests. (default ":9090")
      --log-level string                         log level (default "info")
      --product-info-renewal-interval duration   duration (in go syntax) between renewing the product information. Example: 2h30m (default 24h0m0s)
      --prometheus-address string                http address of a Prometheus instance that has AWS spot price metrics via banzaicloud/spot-price-exporter. If empty, the productinfo app will use current spot prices queried directly from the AWS API.
      --prometheus-query string                  advanced configuration: change the query used to query spot price info from Prometheus. (default "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])")
      --provider strings                         Providers that will be used with the productinfo app. (default [ec2,gce,azure])
```
 
## API calls

*For a complete OpenAPI 3.0 documentation, check out this [URL](https://editor.swagger.io/?url=https://raw.githubusercontent.com/banzaicloud/productinfo/master/docs/openapi/recommender.yaml).*


## FAQ

**1. How do I configure my AWS credentials with the project?**

The project is using the standard [AWS SDK for Go](https://aws.amazon.com/sdk-for-go/), so credentials can be configured via
environment variables, shared credential files and via AWS instance profiles. To learn more about that read the [Specifying Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) section of the SDK docs.

**2. Why do I see messages like `DEBU[0001] Getting available instance types from AWS API. [region=ap-northeast-2, memory=0.5]` when starting the 'productinfo' app?**

After the 'productinfo' app is started, it takes ~2-3 minutes to cache all the product information (like instance types) from AWS (in memory).
AWS is releasing new instance types and regions quite frequently and also changes on-demand pricing from time to time.
So it is necessary to keep this info up-to-date without needing to modify it manually every time something changes on the AWS side.
After the initial query, the 'productinfo' app will parse this info from the AWS Pricing API once per day.
The frequency of this querying and caching is configurable with the `-product-info-renewal-interval` switch and is set to `24h` by default.

**3. What happens if the 'productinfo' app cannot cache the AWS product info?**

If caching fails, the 'productinfo' app will try to reach the AWS Pricing List API on the fly when a request is sent (and it will also cache the resulting information).
If that fails as well, the recommendation will return with an error.

**4. What kind of AWS permissions do I need to use the project?**

The 'productinfo' app is querying the AWS [Pricing API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) to keep up-to-date info
about instance types, regions and on-demand pricing.
You'll need IAM access as described here in [example 11](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-permissions-ref.html#example-policy-pe-api) of the AWS IAM docs.

If you don't use Prometheus to track spot instance pricing, you'll need to be able to access the [spot price history](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotPriceHistory.html) from the AWS API as well with your IAM user.
It means giving permission to `ec2:DescribeSpotPriceHistory`.

**6. What is the advantage of using Prometheus to determine spot prices?**

Prometheus is becoming the de-facto monitoring solution in the cloud native world, and it includes a time series database as well.
When using the Banzai Cloud [spot price exporter](https://github.com/banzaicloud/spot-price-exporter), spot price history will be collected as time series data and
can be queried for averages, maximums and predictions.
It gives a richer picture than relying on the current spot price that can be a spike, or on a downward or upward trend.
You can fine tune your query (with the `-prometheus-query` switch) if you want to change the way spot instance prices are scored.
By default the spot price averages of the last week are queried and instance types are sorted based on this score.

**7. What happens if my Prometheus server cannot be reached or if it doesn't have the necessary spot price metrics?**

If the 'productinfo' app fails to reach the Prometheus query API, or it couldn't find proper metrics, it will fall back to querying the current spot prices from the AWS API.
