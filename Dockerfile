# Build image
FROM golang:1.12.3-alpine AS builder

ENV GOFLAGS="-mod=readonly"

RUN apk add --update --no-cache ca-certificates make git curl mercurial bzr

RUN mkdir -p /workspace
WORKDIR /workspace

ARG GOPROXY

COPY go.* /workspace/
RUN go mod download

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


# UI build image
FROM node:10 as frontend

WORKDIR /web

RUN npm install -g @angular/cli

COPY web/package.json web/package-lock.json /web/

RUN npm install

COPY web/ /web/

RUN ng build --configuration=production --base-href=/


# Final image
FROM alpine:3.9.3

RUN apk add --update --no-cache ca-certificates tzdata bash curl

ARG BUILD_TARGET

RUN if [[ "${BUILD_TARGET}" == "debug" ]]; then apk add --update --no-cache libc6-compat; fi

COPY --from=builder /build/* /usr/local/bin/
COPY --from=frontend /web/dist/ui /web/dist/ui

COPY docker-entrypoint.sh /entrypoint.sh
COPY configs /etc/cloudinfo/serviceconfig

ENV CLOUDINFO_SERVICELOADER_SERVICECONFIGLOCATION "/etc/cloudinfo/serviceconfig"

ENTRYPOINT ["/entrypoint.sh"]
CMD ["cloudinfo"]
