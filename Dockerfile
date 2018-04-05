# Dockerfile for piplelines.linksmart.eu
# Use Dockerfile-multistage to build locally
FROM alpine

RUN apk update && apk add ca-certificates

COPY sample_conf/* /conf/
COPY bin/service-catalog-linux-amd64 /home/

WORKDIR /home
RUN chmod +x service-catalog-linux-amd64

VOLUME /conf /data
EXPOSE 8082

ENV SC_DNSSDENABLED=false
ENV SC_STORAGE_TYPE=leveldb
ENV SC_STORAGE_DSN=/data

ENTRYPOINT ["./service-catalog-linux-amd64"]
CMD ["-conf", "/conf/service-catalog.json"]