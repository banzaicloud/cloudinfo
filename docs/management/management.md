### Cloudinfo Management Service

Cloudinfo contains an internal management api to help administrators operating the service.
The management service is internally exposed on a RESTful API (completely separated from the public API)

These operations mainly affect the Cloud Info Store that backs the Cloudinfo application

The management service cna be configured with the following environment variables:

``management.enabled`` true by default

``management.address`` :8099 by default

If enabled, along with the Cloudinfo application there will be another service started which listens at the address specified in the second env var.

####Management operations:

The context path for management operations is:

<management.address>:/management/store

* Export

    This operation exports the content of the Cloud Product Store
```bash
curl -X PUT \
  http://localhost:8099/management/store/export
```

* Import

    The operation loads data into the CLoud Product Store

```bash
curl -X PUT \
  http://localhost:8099/management/store/export
```

* Refresh
Initiates a scraping process for the given provider for cloud product information. The refresh operation is performed asynchronously so it should only be used to trigger it.
```bash
curl -X PUT \
  http://localhost:8099/management/store/import \
    -H 'cache-control: no-cache' \
  -H 'content-type: multipart/form-data;' \
  -F data=@/Users/puski/data.txt
```


 