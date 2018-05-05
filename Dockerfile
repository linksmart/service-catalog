FROM golang:1.9-alpine as builder

ENV PACKAGE code.linksmart.eu/sc/service-catalog
# copy code
COPY . /home/src/${PACKAGE}

# build
ENV GOPATH /home
RUN go install ${PACKAGE}

###########
FROM alpine

RUN apk --no-cache add ca-certificates

WORKDIR /home
COPY --from=builder /home/bin/* .
COPY sample_conf/* /conf/

ENV SC_DNSSDENABLED=false
ENV SC_STORAGE_TYPE=leveldb
ENV SC_STORAGE_DSN=/data

VOLUME /conf /data
EXPOSE 8082

ENTRYPOINT ["./service-catalog"]
CMD ["-conf", "/conf/service-catalog.json"]