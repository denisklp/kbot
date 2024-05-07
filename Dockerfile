FROM quay.io/projectquay/golang:1.22 as builder

WORKDIR /go/src/app
COPY . .
ARG TARGETOS=linux TARGETARCH=amd64 VERSION=v1.0.0
#RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o kbot -ldflags "-X="github.com/denisklp/kbot/cmd.appVersion=${VERSION}
RUN make build TARGETOS=$TARGETOS TARGETARCH=$TARGETARCH VERSION=$VERSION


FROM scratch
WORKDIR /
COPY --from=builder /go/src/app/kbot .
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["./kbot"]