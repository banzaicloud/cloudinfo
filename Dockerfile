
FROM golang:1.11-alpine as backend
RUN apk add --update --no-cache bash ca-certificates curl git make tzdata

RUN mkdir -p /go/src/github.com/banzaicloud/cloudinfo
ADD Gopkg.* Makefile main-targets.mk /go/src/github.com/banzaicloud/cloudinfo/
WORKDIR /go/src/github.com/banzaicloud/cloudinfo

RUN make vendor
ADD . /go/src/github.com/banzaicloud/cloudinfo

RUN make build

FROM node:10 as frontend
ADD ./web /web
WORKDIR /web
RUN npm install
RUN npm install -g @angular/cli
RUN ng build --configuration=production --base-href=/

FROM alpine:3.7
RUN apk add --update --no-cache bash curl

COPY --from=backend /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend /go/src/github.com/banzaicloud/cloudinfo/build/cloudinfo /bin
COPY --from=frontend /web/dist/ui /web/dist/ui
ADD ./entrypoint.sh /entrypoint.sh
ADD ./configs /configs

ENV CLOUDINFO_BASEPATH "/cloudinfo"
ENV SERVICELOADER_SERVICECONFIGLOCATION "/configs"

ENTRYPOINT ["/entrypoint.sh"]
