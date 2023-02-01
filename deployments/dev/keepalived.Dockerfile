FROM alpine:latest

RUN apk --update add keepalived

ENTRYPOINT [ "keepalived" ]
CMD [ "-nlD" ]
