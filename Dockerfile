FROM golang:alpine

RUN apk add --no-cache make git bash

WORKDIR /keepalived-exporter

ADD . .

RUN make build

EXPOSE 9165

ENTRYPOINT [ "./keepalived-exporter" ]
