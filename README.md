
[![CI](https://github.com/banzaicloud/cloudinfo/actions/workflows/ci.yaml/badge.svg)](https://github.com/banzaicloud/cloudinfo/actions/workflows/ci.yaml)
[![Docker](https://github.com/banzaicloud/cloudinfo/actions/workflows/docker.yaml/badge.svg)](https://github.com/banzaicloud/cloudinfo/actions/workflows/docker.yaml)
[![Helm chart](https://github.com/banzaicloud/cloudinfo/actions/workflows/helm_chart.yaml/badge.svg)](https://github.com/banzaicloud/cloudinfo/actions/workflows/helm_chart.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/cloudinfo)](https://goreportcard.com/report/github.com/banzaicloud/cloudinfo)
![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1651/badge)](https://bestpractices.coreinfrastructure.org/projects/1651)


# Cloud price and service information

The Banzai Cloud Cloudinfo application is a standalone project in the [Pipeline](https://github.com/banzaicloud/pipeline) ecosystem.
While AWS, Google Cloud, Azure, AliBaba or Oracle all provide some kind of APIs to query instance type attributes and product pricing information, these APIs are often responding with partly inconsistent data, or the responses are very cumbersome to parse.
The Cloudinfo service uses these cloud provider APIs to asynchronously fetch and parse instance type attributes and prices, while storing the results in an in memory cache and making it available as structured data through a REST API.

## Feature Testing
1. `make docker` (without args) # on your feature branch
2. `make docker-push` # to push image created in previous step to Harness docker repository `gcr.io/platform-205701/banzaicloud/cloudinfo:docker-tag`

## Release Process
1. `git tag 0.18.0 -a` # create a local tag on master branch
2. `git push origin 0.18.0` # push the tag information to github
3. `make docker docker-push` # build docker image locally and push to GCR
4. Create a new release page with `0.18.0`, it will automatically create changeset.


## Update swagger specs
1. `make bin/swagger` # Make sure swagger executable is present
2. `make swagger` # to update swagger specs present in `api/openapi-spec/cloudinfo.*`

## Quick start

Building the project is as simple as running a go build command. The result is a statically linked executable binary.

```
make build
```

The following options can be configured when starting the exporter (with defaults):

```
build/cloudinfo --help
Usage of Banzai Cloud Cloudinfo Service:
      --config-vault string               enable config Vault
      --config-vault-address string       config Vault address
      --config-vault-token string         config Vault token
      --config-vault-secret-path string   config Vault secret path
      --log-level string                  log level (default "info")
      --log-format string                 log format (default "json")
      --metrics-enabled                   internal metrics are exposed if enabled
      --metrics-address string            the address where internal metrics are exposed (default ":9090")
      --listen-address string             application listen address (default ":8000")
      --scrape                            enable cloud info scraping (default true)
      --scrape-interval duration          duration (in go syntax) between renewing information (default 24h0m0s)
      --provider-amazon                   enable amazon provider
      --provider-google                   enable google provider
      --provider-alibaba                  enable alibaba provider
      --provider-oracle                   enable oracle provider
      --provider-azure                    enable azure provider
      --provider-digitalocean             enable digitalocean provider
      --config string                     Configuration file
      --version                           Show version information
      --dump-config                       Dump configuration to the console (and exit)
```

Create a permanent developer configuration:

```bash
cp config.toml.dist config.toml
```

Running `cloudinfo` requires the `web/` project to be built (requires `Node.js` to be installed):

```bash
cd web/
npm run build-prod
cd ..
build/cloudinfo
```

## Cloud credentials

The cloudinfo service is querying the cloud provider APIs, so it needs credentials to access these.

### AWS

Cloudinfo is using the AWS [Price List API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) that allows a user to query product pricing in a fine-grained way.
Authentication works through the standard [AWS SDK for Go](https://aws.amazon.com/sdk-for-go/), so credentials can be configured via
environment variables, shared credential files and via AWS instance profiles. To learn more about that read the [Specifying Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) section of the SDK docs.

The easiest way is through environment variables:

```
export AWS_ACCESS_KEY_ID=<access-key-id>
export AWS_SECRET_ACCESS_KEY=<secret-access-key>
cloudinfo --provider-amazon
```

Create AWS credentials with aws command-line tool:

```
aws iam create-user --user-name cloudinfo
aws iam put-user-policy --user-name cloudinfo --policy-name cloudinfo_policy --policy-document file://credentials/amazon_cloudinfo_role.json
aws iam create-access-key --user-name cloudinfo
```

### Google Cloud

On Google Cloud the project is using two different APIs to collect the full product information: the Cloud Billing API and the Compute Engine API.
Authentication to the `Cloud Billing Catalog API` is done through an [API key](https://cloud.google.com/docs/authentication/api-keys) that can be generated on the [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
Once you have an API key, billing is [enabled](https://cloud.google.com/billing/docs/how-to/modify-project) for the project and the Cloud Billing API is also [enabled](https://console.cloud.google.com/flows/enableapi?apiid=cloudbilling.googleapis.com) you can start using the API.

The `Compute Engine API` is doing authentication in the standard Google Cloud way with [service accounts](https://cloud.google.com/compute/docs/access/service-accounts) instead of API keys.
Once you have a service account, download the JSON credentials file from the Google Cloud Console, and set its account through an environment variable:

```
export GOOGLE_CREDENTIALS_FILE=<path-to-my-service-account-file>.json
export GOOGLE_PROJECT=<google-project-id>
cloudinfo --provider-google
```

Create service account key with gcloud command-line tool:

```
gcloud services enable container.googleapis.com compute.googleapis.com cloudbilling.googleapis.com cloudresourcemanager.googleapis.com
gcloud iam service-accounts create cloudinfoSA --display-name "Service account used for managing Cloudinfo”
gcloud iam roles create cloudinfo --project [PROJECT-ID] --title cloudinfo --description "cloudinfo roles" --permissions compute.machineTypes.list,compute.regions.list,compute.zones.list
gcloud projects add-iam-policy-binding [PROJECT-ID] --member='serviceAccount:cloudinfoSA@[PROJECT-ID].iam.gserviceaccount.com' --role='projects/[PROJECT-ID]/roles/cloudinfo'
gcloud iam service-accounts keys create cloudinfo.gcloud.json --iam-account=cloudinfoSA@[PROJECT-ID].iam.gserviceaccount.com
```

### Azure

There are two different APIs used for Azure that provide machine type information and SKUs respectively.
Pricing info can be queried through the [Rate Card API](https://msdn.microsoft.com/en-us/library/azure/mt219004).
Machine types can be queried through the Compute API's [list virtual machine sizes](https://docs.microsoft.com/en-us/rest/api/compute/virtualmachinesizes/list) request.
Authentication is done via standard Azure service principals.

Follow [this](https://docs.microsoft.com/en-us/go/azure/azure-sdk-go-qs-vm#create-a-service-principal) link to learn how to generate one with the Azure SDK
and set an environment variable that points to the service account file:

```
export AZURE_SUBSCRIPTION_ID=<subscription-id>
export AZURE_TENANT_ID=<tenant-id>
export AZURE_CLIENT_ID=<client-id>
export AZURE_CLIENT_SECRET=<client-secret>
cloudinfo --provider-azure
```

Create service principal with azure command-line tool:

```
cd credentials
az provider register --namespace Microsoft.Compute
az provider register --namespace Microsoft.Resources
az provider register --namespace Microsoft.ContainerService
az provider register --namespace Microsoft.Commerce
az role definition create --verbose --role-definition @azure_cloudinfo_role.json
az ad sp create-for-rbac --name "CloudinfoSP" --role "Cloudinfo" --sdk-auth true > azure_cloudinfo.auth
```

### Oracle

Authentication is done via CLI configuration file. Follow [this](https://docs.cloud.oracle.com/iaas/Content/API/Concepts/sdkconfig.htm) link to learn how to create such a file and set an environment variable that points to that config file:

```
export ORACLE_TENANCY_OCID=<tenancy-ocid>
export ORACLE_USER_OCID=<user-ocid>
export ORACLE_REGION=<region>
export ORACLE_FINGERPRINT=<fingerprint>
export ORACLE_PRIVATE_KEY=<private-key>
export ORACLE_PRIVATE_KEY_PASSPHRASE=<private-key-passphrase>
# OR
export ORACLE_CONFIG_FILE_PATH=<config-file-path>
export ORACLE_PROFILE=<profile>

cloudinfo --provider-oracle
```

### Alibaba

The easiest way to authenticate is through environment variables:

```
export ALIBABA_ACCESS_KEY_ID=<access-key-id>
export ALIBABA_ACCESS_KEY_SECRET=<access-key-secret>
export ALIBABA_REGION_ID=<region-id>
cloudinfo --provider-alibaba
```

Create Alibaba credentials with Alibaba Cloud CLI:

```
aliyun ram CreateUser --UserName CloudInfo --DisplayName CloudInfo
aliyun ram AttachPolicyToUser --UserName CloudInfo --PolicyName AliyunECSReadOnlyAccess --PolicyType System
aliyun ram AttachPolicyToUser --UserName CloudInfo --PolicyName AliyunBSSReadOnlyAccess --PolicyType System
aliyun ram CreateAccessKey --UserName CloudInfo
```

### DigitalOcean

```
export DIGITALOCEAN_ACCESS_TOKEN=<access-token>
cloudinfo --provider-digitalocean
```

Create a new API access token on [DigitalOcean Console](https://cloud.digitalocean.com/account/api/tokens).

### Configuring multiple providers

Cloud providers can be configured one by one. To configure multiple providers simply list all of them and configure the credentials for all of them.
Here's an example of how to configure three providers:
```
export AWS_SECRET_ACCESS_KEY=<secret-access-key>
export AWS_ACCESS_KEY_ID=<access-key-id>
export ALIBABA_ACCESS_KEY_ID=<access-key-id>
export ALIBABA_ACCESS_KEY_SECRET=<access-key-secret>
export ALIBABA_REGION_ID=<region-id>
export DIGITALOCEAN_ACCESS_TOKEN=<access-token>

cloudinfo --provider-amazon --provider-alibaba --provider-digitalocean
```

## API calls

*For a complete OpenAPI 3.0 documentation, check out this [URL](https://editor.swagger.io/?url=https://raw.githubusercontent.com/banzaicloud/cloudinfo/master/api/openapi-spec/cloudinfo.yaml).*

Here's a few `cURL` examples to get started:

```
curl  -ksL -X GET "http://localhost:9090/api/v1/providers/azure/services/compute/regions/" | jq .
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
curl  -ksL -X GET "http://localhost:9090/api/v1/providers/amazon/services/compute/regions/eu-west-1/products" | jq .
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

**1. The API responses with status code 500 after starting the `cloudinfo` app and making a `cURL` request**

After the `cloudinfo` app is started, it takes a few minutes to cache all the product information from the providers.
Before the results are cached, responses may be unreliable. We're planning to solve it in the future. After a few minutes it should work fine.

**2. Why is it needed to parse the product info asynchronously and periodically instead of relying on static data?**

Cloud providers are releasing new instance types and regions quite frequently and also changing on-demand pricing from time to time.
So it is necessary to keep this info up-to-date without needing to modify it manually every time something changes on the provider's side.
After the initial query, the `cloudinfo` app will parse this info from the Cloud providers once per day.
The frequency of this querying and caching is configurable with the `--product-info-renewal-interval` switch and is set to `24h` by default.

**3. What happens if the `cloudinfo` app cannot cache the AWS product info?**

If caching fails, the `cloudinfo` app will try to reach the AWS Pricing List API on the fly when a request is sent (and it will also cache the resulting information).

**4. What kind of AWS permissions do I need to use the project?**

The `cloudinfo` app is querying the AWS [Pricing API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) to keep up-to-date info
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

If the `cloudinfo` app fails to reach the Prometheus query API, or it couldn't find proper metrics, it will fall back to querying the current spot prices from the AWS API.

### License

Copyright (c) 2017-2019 [Banzai Cloud, Inc.](https://banzaicloud.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
