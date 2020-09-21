FROM golang:alpine

RUN apk --update -t add keepalived git

RUN go get -u github.com/msoap/shell2http

COPY entrypoint.sh entrypoint.sh

EXPOSE 8080

ENTRYPOINT [ "./entrypoint.sh" ]
