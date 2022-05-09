FROM golang:1.18 as builder
 
WORKDIR /go/src/github.com/automatedhome/circulation
COPY . .
RUN CGO_ENABLED=0 go build -o circulation cmd/main.go

FROM busybox:glibc

COPY --from=builder /go/src/github.com/automatedhome/circulation/circulation /usr/bin/circulation

HEALTHCHECK --timeout=5s --start-period=1m \
  CMD wget --quiet --tries=1 --spider http://localhost:7003/health || exit 1

EXPOSE 7003
ENTRYPOINT [ "/usr/bin/circulation" ]
