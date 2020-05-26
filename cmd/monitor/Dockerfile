FROM alpine:latest

RUN mkdir -p /app/configs /app/db

ADD configs/config.yml /app/configs/
ADD configs/abi/ /app/abi/
ADD db/ /app/db/
ADD shared-db/ /app/shared-db/

ADD https://github.com/amacneil/dbmate/releases/download/v1.7.0/dbmate-linux-musl-amd64 /usr/bin/dbmate
ADD bin/monitor /app/
ADD cmd/monitor/entrypoint.sh /app/

RUN chmod a+x /usr/bin/dbmate /app/monitor /app/entrypoint.sh

WORKDIR /app
CMD ["./entrypoint.sh"]
