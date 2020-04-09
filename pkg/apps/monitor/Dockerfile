FROM golang:1.13.4 AS builder
RUN go version
WORKDIR /app
COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o app .

FROM scratch
WORKDIR /root/
COPY --from=builder /app/app .
EXPOSE 8888
ENTRYPOINT ["./app"]