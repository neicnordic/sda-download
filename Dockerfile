FROM golang:alpine as builder

ENV GOPATH=$PWD
ENV CGO_ENABLED=0

COPY . .

RUN go build -buildvcs=false -o ./sda-download ./cmd
RUN echo "nobody:x:65534:65534:nobody:/:/sbin/nologin" > passwd

# build middleware plugins
WORKDIR api/middleware/plugins
RUN apk update && apk add bash build-base
RUN GOPATH=$PWD CGO_ENABLED=1 ./build.sh

FROM scratch

ARG BUILD_DATE
ARG SOURCE_COMMIT

LABEL maintainer="NeIC System Developers"
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$BUILD_DATE
LABEL org.label-schema.vcs-url="https://github.com/neicnordic/sda-download"
LABEL org.label-schema.vcs-ref=$SOURCE_COMMIT

COPY --from=builder /go/passwd /etc/passwd
COPY --from=builder /go/sda-* /usr/bin/
COPY --from=builder /go/api/middleware/plugins/*.so /plugins/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 65534
