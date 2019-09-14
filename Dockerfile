FROM golang:1.13-alpine as builder

COPY . /home

WORKDIR /home
ENV CGO_ENABLED=0

ARG version
ARG buildnum
RUN go build -v -mod=vendor -o service-catalog \
        -ldflags "-X main.Version=$version -X main.BuildNumber=$buildnum"

###########
FROM alpine

RUN apk --no-cache add ca-certificates

ARG version
ARG buildnum
LABEL NAME="LinkSmart Service Catalog"
LABEL VERSION=${version}
LABEL BUILD=${buildnum}

WORKDIR /home
COPY --from=builder /home/service-catalog .
COPY sample_conf/* /conf/

ENV SC_DNSSDENABLED=false
ENV SC_STORAGE_TYPE=leveldb
ENV SC_STORAGE_DSN=/data

VOLUME /conf /data
EXPOSE 8082

ENTRYPOINT ["./service-catalog"]
CMD ["-conf", "/conf/service-catalog.json"]
