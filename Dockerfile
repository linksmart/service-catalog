FROM debian:stable-slim

COPY sample_conf/* /conf/
COPY bin /home

WORKDIR /home
RUN chmod +x service-catalog-linux-amd64

VOLUME /conf /data
EXPOSE 8082

ENTRYPOINT ["./service-catalog-linux-amd64"]
CMD ["-conf", "/conf/docker.json"]
