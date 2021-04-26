FROM golang:1.14 AS builder

WORKDIR /root/jsonbox_exporter
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 GOOS=linux go build -a -trimpath -o bin/jsonbox_exporter cmd/main.go

FROM alpine:3.12

LABEL maintainer="LyleLaii"

WORKDIR /opt/jsonbox_exporter

COPY --from=builder /root/jsonbox_exporter/bin /opt/jsonbox_exporter/bin
COPY example/config.yml /opt/jsonbox_exporter/etc/config.yml

EXPOSE 7979

ENTRYPOINT [ "bin/jsonbox_exporter" ]