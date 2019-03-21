FROM golang:1.12-alpine as builder

COPY . /home

WORKDIR /home
RUN go build -mod=vendor -o service-catalog

###########
FROM alpine

RUN apk --no-cache add ca-certificates

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