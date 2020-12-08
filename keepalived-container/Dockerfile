ARG GOVERSION=1.14

FROM golang:${GOVERSION}-alpine as builder

RUN apk add --no-cache git

RUN CGO_ENABLED=0 go get -u github.com/msoap/shell2http

FROM alpine:3.12

RUN apk --update -t add --no-cache keepalived

COPY --from=builder /go/bin/shell2http /usr/local/bin/shell2http

COPY entrypoint.sh .

EXPOSE 8080

ENTRYPOINT [ "./entrypoint.sh" ]
