FROM golang:1.12.3-alpine AS builder

ENV GOFLAGS="-mod=readonly"

RUN apk add --update --no-cache ca-certificates make git curl mercurial bzr

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN BINARY_NAME=cloudinfo make build-release


FROM node:10 as frontend

WORKDIR /web

COPY web /web

RUN npm install
RUN npm install -g @angular/cli
RUN ng build --configuration=production --base-href=/


FROM alpine:3.9.3

RUN apk add --update --no-cache ca-certificates tzdata bash curl

COPY --from=builder /build/build/release/cloudinfo /bin
COPY --from=frontend /web/dist/ui /web/dist/ui

COPY entrypoint.sh /entrypoint.sh
COPY configs /configs

ENV CLOUDINFO_BASEPATH "/cloudinfo"
ENV SERVICELOADER_SERVICECONFIGLOCATION "/configs"

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/cloudinfo"]
