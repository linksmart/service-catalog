FROM alpine

COPY sample_conf/* /conf/
COPY bin/service-catalog-linux-amd64 /home/

WORKDIR /home
RUN chmod +x service-catalog-linux-amd64

VOLUME /conf /data
EXPOSE 8082

ENTRYPOINT ["./service-catalog-linux-amd64"]
CMD ["-conf", "/conf/docker.json"]