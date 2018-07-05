# build stage
FROM golang:1.9.3-alpine3.7

ADD . /go/src/github.com/banzaicloud/productinfo
WORKDIR /go/src/github.com/banzaicloud/productinfo
RUN go build -o /bin/productinfo ./cmd/productinfo

FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=0 /bin/productinfo /bin
ENTRYPOINT ["/bin/productinfo"]
