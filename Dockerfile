FROM alpine:latest

RUN mkdir -p /app/bin

ADD contract.abi event.abi /app/
ADD platform-action-monitor /app/bin/

WORKDIR /app/bin

CMD  ["./platform-action-monitor"]
