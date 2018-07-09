FROM golang:1.9.3-alpine3.7 as backend
ADD . /go/src/github.com/banzaicloud/productinfo
WORKDIR /go/src/github.com/banzaicloud/productinfo
RUN go build -o /bin/productinfo ./cmd/productinfo

FROM node:9 as frontend
ADD ./web /web
WORKDIR /web
RUN npm install
RUN npm install -g @angular/cli
RUN ng build --configuration=production --base-href=/productinfo/

FROM alpine:3.7
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=backend /bin/productinfo /bin
COPY --from=frontend /web/dist/ui /web/dist/ui

ENV PRODUCTINFO_BASEPATH "/productinfo"

ENTRYPOINT ["/bin/productinfo"]
