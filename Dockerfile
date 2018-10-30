
FROM golang:1.11-alpine as backend
RUN apk add --update --no-cache bash ca-certificates curl git make tzdata

RUN mkdir -p /go/src/github.com/banzaicloud/productinfo
ADD Gopkg.* Makefile main-targets.mk /go/src/github.com/banzaicloud/productinfo/
WORKDIR /go/src/github.com/banzaicloud/productinfo

RUN make vendor
ADD . /go/src/github.com/banzaicloud/productinfo

RUN make build

FROM node:9 as frontend
ADD ./web /web
WORKDIR /web
RUN npm install
RUN npm install -g @angular/cli
RUN ng build --configuration=production --base-href=/productinfo/

FROM alpine:3.7
COPY --from=backend /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend /go/src/github.com/banzaicloud/productinfo/build/productinfo /bin
COPY --from=frontend /web/dist/ui /web/dist/ui

ENV PRODUCTINFO_BASEPATH "/productinfo"

ENTRYPOINT ["/bin/productinfo"]
