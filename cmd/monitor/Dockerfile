FROM alpine:latest

RUN mkdir -p /app

ADD configs/ /app/configs/
ADD bin/monitor /app/

RUN chmod a+x /app/monitor

WORKDIR /app

CMD  ["./monitor","-config","/app/configs/config.yml"]
