FROM golang:1.15-alpine AS build

ADD . controller

RUN cd controller && \
    CGO_ENABLED=0 go build . && \
    apk --no-cache add binutils && \
    strip -s thermo-center-controller

FROM scratch

COPY --from=build /go/controller/thermo-center-controller /

USER 21586

CMD ["/thermo-center-controller"]
