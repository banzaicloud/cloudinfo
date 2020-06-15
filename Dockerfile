# UI build image
FROM node:12.3.1 as frontend

WORKDIR /web

COPY web/package.json web/package-lock.json /web/

RUN npm install

COPY web/ /web/

RUN npm run build-prod


# Build image
FROM golang:1.14-alpine3.11 AS builder

ENV GOFLAGS="-mod=readonly"

RUN apk add --update --no-cache ca-certificates make git curl mercurial bzr

RUN mkdir -p /workspace
WORKDIR /workspace

ARG GOPROXY

COPY go.* /workspace/
RUN go mod download

COPY Makefile main-targets.mk /workspace/

COPY --from=frontend /web/dist/web /workspace/web/dist/web
COPY . /workspace

ARG BUILD_TARGET

RUN set -xe && \
    if [[ "${BUILD_TARGET}" == "debug" ]]; then \
        cd /tmp; GOBIN=/workspace/build/debug go get github.com/go-delve/delve/cmd/dlv; cd -; \
        make build-debug; \
        mv build/debug /build; \
    else \
        make build-release; \
        mv build/release /build; \
    fi


# Final image
FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates tzdata bash curl

SHELL ["/bin/bash", "-c"]

# set up nsswitch.conf for Go's "netgo" implementation
# https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-424546457
RUN test ! -e /etc/nsswitch.conf && echo 'hosts: files dns' > /etc/nsswitch.conf

ARG BUILD_TARGET

RUN if [[ "${BUILD_TARGET}" == "debug" ]]; then apk add --update --no-cache libc6-compat; fi

COPY --from=builder /build/* /usr/local/bin/

COPY configs /etc/cloudinfo/serviceconfig

RUN sed -i "s|dataLocation: ./configs/|dataLocation: /etc/cloudinfo/serviceconfig/|g" /etc/cloudinfo/serviceconfig/services.yaml

ENV CLOUDINFO_SERVICELOADER_SERVICECONFIGLOCATION "/etc/cloudinfo/serviceconfig"

CMD ["cloudinfo"]
