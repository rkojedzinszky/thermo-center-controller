FROM golang:1.15-alpine3.12 AS build

ADD . controller

RUN cd controller && \
    CGO_ENABLED=0 go build . && \
    apk --no-cache add binutils && \
    strip -s thermo-center-controller

FROM scratch

LABEL org.opencontainers.image.authors "Richard Kojedzinszky <richard@kojedz.in>"
LABEL org.opencontainers.image.source https://github.com/rkojedzinszky/thermo-center-controller

COPY --from=build /go/controller/thermo-center-controller /

USER 21586

CMD ["/thermo-center-controller"]
