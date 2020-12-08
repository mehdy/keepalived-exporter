ARG GOVERSION=1.14

FROM golang:${GOVERSION}-alpine as builder

WORKDIR /build

RUN apk add --no-cache make git bash

ADD . .

RUN make build

FROM scratch 

COPY --from=builder /build/keepalived-exporter . 

EXPOSE 9165

ENTRYPOINT [ "./keepalived-exporter" ]
