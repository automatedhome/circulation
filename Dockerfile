FROM arm32v7/golang:stretch

COPY qemu-arm-static /usr/bin/
WORKDIR /go/src/github.com/automatedhome/circulation
COPY . .
RUN make build

FROM arm32v7/busybox:1.30-glibc

COPY --from=0 /go/src/github.com/automatedhome/circulation/circulation /usr/bin/circulation

HEALTHCHECK --timeout=5s --start-period=1m \
  CMD wget --quiet --tries=1 --spider http://localhost:7003/health || exit 1

EXPOSE 7003
ENTRYPOINT [ "/usr/bin/circulation" ]
