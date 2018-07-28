
[![CircleCI](https://circleci.com/gh/banzaicloud/productinfo/tree/master.svg?style=shield)](https://circleci.com/gh/banzaicloud/productinfo/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/productinfo)](https://goreportcard.com/report/github.com/banzaicloud/productinfo)
![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1651/badge)](https://bestpractices.coreinfrastructure.org/projects/1651)


# Product Info

The Banzai Cloud `productinfo` application is a standalone project in the [Pipeline](https://github.com/banzaicloud/pipeline) ecosystem.
While EC2, Google Cloud and Azure all provide some kind of APIs to query instance type attributes and product pricing information, these APIs are often responding with partly inconsistent data, or the responses are very cumbersome to parse.
The Productinfo service uses these cloud provider APIs to asynchronously fetch and parse instance type attributes and prices, while storing the results in an in memory cache and making it available as structured data through a REST API.
See the UI in action here: [https://banzaicloud.com/productinfo/](https://banzaicloud.com/productinfo/)

## Quick start

Building the project is as simple as running a go build command. The result is a statically linked executable binary.

```
go build ./cmd/productinfo
```

The following options can be configured when starting the exporter (with defaults):

```
./productinfo --help
Usage of ./productinfo:
      --azure-subscription-id string             Azure subscription ID to use with the APIs
      --gce-api-key string                       GCE API key to use for getting SKUs
      --help                                     print usage
      --listen-address string                    the address the productinfo app listens to HTTP requests. (default ":9090")
      --log-level string                         log level (default "info")
      --product-info-renewal-interval duration   duration (in go syntax) between renewing the product information. Example: 2h30m (default 24h0m0s)
      --prometheus-address string                http address of a Prometheus instance that has AWS spot price metrics via banzaicloud/spot-price-exporter. If empty, the productinfo app will use current spot prices queried directly from the AWS API.
      --prometheus-query string                  advanced configuration: change the query used to query spot price info from Prometheus. (default "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])")
      --provider strings                         Providers that will be used with the productinfo application. (default [ec2,gce,azure,oracle])
```

## Cloud credentials

The Productinfo service is querying the cloud provider APIs, so it needs credentials to access these.

### AWS

Productinfo is using the AWS [Price List API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) that allows a user to query product pricing in a fine-grained way.
Authentication works through the standard [AWS SDK for Go](https://aws.amazon.com/sdk-for-go/), so credentials can be configured via
environment variables, shared credential files and via AWS instance profiles. To learn more about that read the [Specifying Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) section of the SDK docs.

The easiest way is through environment variables:

```
export AWS_SECRET_ACCESS_KEY=<your-secret-access-key>
export AWS_ACCESS_KEY_ID=<your-access-key-id>
./productinfo --provider ec2
```

### Google Cloud

On Google Cloud the project is using two different APIs to collect the full product information: the Cloud Billing API and the Compute Engine API.
Authentication to the `Cloud Billing Catalog API` is done through an [API key](https://cloud.google.com/docs/authentication/api-keys) that can be generated on the [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
Once you have an API key, billing is [enabled](https://cloud.google.com/billing/docs/how-to/modify-project) for the project and the Cloud Billing API is also [enabled](https://console.cloud.google.com/flows/enableapi?apiid=cloudbilling.googleapis.com) you can start using the API.

The `Compute Engine API` is doing authentication in the standard Google Cloud way with [service accounts](https://cloud.google.com/compute/docs/access/service-accounts) instead of API keys.
Once you have a service account, download the JSON credentials file from the Google Cloud Console, and set its account through an environment variable:

```
export GOOGLE_APPLICATION_CREDENTIALS=<path-to-my-service-account-file>.json
./productinfo --provider gce --gce-api-key "<gce-api-key>"

```

### Azure

There are two different APIs used for Azure that provide machine type information and SKUs respectively.
Pricing info can be queried through the [Rate Card API](https://msdn.microsoft.com/en-us/library/azure/mt219004).
Machine types can be queried through the Compute API's [list virtual machine sizes](https://docs.microsoft.com/en-us/rest/api/compute/virtualmachinesizes/list) request.
Authentication is done via standard Azure service principals.

Follow [this](https://docs.microsoft.com/en-us/go/azure/azure-sdk-go-qs-vm#create-a-service-principal) link to learn how to generate one with the Azure SDK
and set an environment variable that points to the service account file:

```
export AZURE_AUTH_LOCATION=<path-to-service-principal>.auth
./productinfo --provider azure --azure-subscription-id "ba96ef31-4a42-40f5-8740-03f7e3c439eb"
```

### Oracle

Authentication is done via CLI configuration file. Follow [this](https://docs.cloud.oracle.com/iaas/Content/API/Concepts/sdkconfig.htm) link to learn how to create such a file and set an environment variable that points to that config file:

```
export ORACLE_CLI_CONFIG_LOCATION=<path-to-oci-cli-configuration>
./productinfo --provider oracle
```

### Configuring multiple providers

Cloud providers can be configured one by one. To configure multiple providers simply list all of them and configure the credentials for all of them.
Here's an example of how to configure all three providers:
```
export AWS_SECRET_ACCESS_KEY=<your-secret-access-key>
export AWS_ACCESS_KEY_ID=<your-access-key-id>
export GOOGLE_APPLICATION_CREDENTIALS=<path-to-my-service-account-file>.json
export AZURE_AUTH_LOCATION=<path-to-service-principal>.auth
export ORACLE_CLI_CONFIG_LOCATION=<path-to-oci-cli-configuration>
./productinfo --provider ec2 --provider gce --gce-api-key "<gce-api-key>" --provider azure --azure-subscription-id "ba96ef31-4a42-40f5-8740-03f7e3c439eb" --provider oracle

```

## API calls

*For a complete OpenAPI 3.0 documentation, check out this [URL](https://editor.swagger.io/?url=https://raw.githubusercontent.com/banzaicloud/productinfo/master/api/openapi-spec/productinfo.yaml).*

Here's a few `cURL` examples to get started:

```
curl  -ksL -X GET "http://localhost:9091/api/v1/regions/azure/" | jq .
[
  {
    "id": "centralindia",
    "name": "Central India"
  },
  {
    "id": "koreacentral",
    "name": "Korea Central"
  },
  {
    "id": "southindia",
    "name": "South India"
  },
  ...
]
```

```
curl  -ksL -X GET "http://localhost:9091/api/v1/products/ec2/eu-west-1" | jq .
{
  "products": [
    {
      "type": "i3.8xlarge",
      "onDemandPrice": 2.752,
      "cpusPerVm": 32,
      "memPerVm": 244,
      "gpusPerVm": 0,
      "ntwPerf": "10 Gigabit",
      "ntwPerfCategory": "high",
      "spotPrice": [
        {
          "zone": "eu-west-1c",
          "price": 1.6018
        },
        {
          "zone": "eu-west-1b",
          "price": 0.9563
        },
        {
          "zone": "eu-west-1a",
          "price": 2.752
        }
      ]
    },
    ...
  ]
}
```

## FAQ

**1. The API responses with status code 500 after starting the `productinfo` app and making a `cURL` request**

After the `productinfo` app is started, it takes a few minutes to cache all the product information from the providers.
Before the results are cached, responses may be unreliable. We're planning to solve it in the future. After a few minutes it should work fine.

**2. Why is it needed to parse the product info asynchronously and periodically instead of relying on static data?**

Cloud providers are releasing new instance types and regions quite frequently and also changing on-demand pricing from time to time.
So it is necessary to keep this info up-to-date without needing to modify it manually every time something changes on the provider's side.
After the initial query, the `productinfo` app will parse this info from the Cloud providers once per day.
The frequency of this querying and caching is configurable with the `--product-info-renewal-interval` switch and is set to `24h` by default.

**3. What happens if the `productinfo` app cannot cache the AWS product info?**

If caching fails, the `productinfo` app will try to reach the AWS Pricing List API on the fly when a request is sent (and it will also cache the resulting information).

**4. What kind of AWS permissions do I need to use the project?**

The `productinfo` app is querying the AWS [Pricing API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) to keep up-to-date info
about instance types, regions and on-demand pricing.
You'll need IAM access as described here in [example 11](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-permissions-ref.html#example-policy-pe-api) of the AWS IAM docs.

If you don't use Prometheus to track spot instance pricing, you'll need to be able to access the [spot price history](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotPriceHistory.html) from the AWS API as well with your IAM user.
It means giving permission to `ec2:DescribeSpotPriceHistory`.

**5. What is the advantage of using Prometheus to determine spot prices?**

Prometheus is becoming the de-facto monitoring solution in the cloud native world, and it includes a time series database as well.
When using the Banzai Cloud [spot price exporter](https://github.com/banzaicloud/spot-price-exporter), spot price history will be collected as time series data and
can be queried for averages, maximums and predictions.
It gives a richer picture than relying on the current spot price that can be a spike, or on a downward or upward trend.
You can fine tune your query (with the `-prometheus-query` switch) if you want to change the way spot instance prices are scored.
By default the spot price averages of the last week are queried and instance types are sorted based on this score.

**6. What happens if my Prometheus server cannot be reached or if it doesn't have the necessary spot price metrics?**

If the `productinfo` app fails to reach the Prometheus query API, or it couldn't find proper metrics, it will fall back to querying the current spot prices from the AWS API.
