FROM alpine:latest

RUN mkdir -p /app/configs

ADD configs/config.yml /app/configs/
ADD configs/abi/ /app/abi/
ADD bin/monitor /app/

RUN chmod a+x /app/monitor

WORKDIR /app

CMD  ["./monitor","-config","/app/configs/config.yml"]
