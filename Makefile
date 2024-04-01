APP := $(shell basename $(shell git remote get-url origin))
REGISTRY := mrfwdmail
VERSION=$(shell git describe --tags --abbrev=0)-$(shell git rev-parse --short HEAD)
TARGETOS=linux #linux darwin windows
TARGETARCH=amd64 #arm64

format:
	gofmt -s -w ./

lint:
	golint

test:
	go test -v

get:
	go get

build: format get
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o kbot -ldflags "-X="github.com/mrgitmail/kbot/cmd.appVersion=${VERSION}

image:
	docker build . -t ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}-${TARGETOS} --build-arg TARGETARCH=${TARGETARCH} --build-arg TARGETOS=${TARGETOS}

push:
	docker push ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}-${TARGETOS}

clean:
	rm -rf kbot
	docker rmi ${REGISTRY}/${APP}:${VERSION}-${TARGETARCH}-${TARGETOS}

linux:
	$(MAKE) build TARGETOS=linux TARGETARCH=amd64


darwin:
	$(MAKE) build TARGETOS=darwin TARGETARCH=amd64


windows:
	$(MAKE) build TARGETOS=windows TARGETARCH=amd64


arm:
	$(MAKE) build TARGETOS=linux TARGETARCH=arm64
	