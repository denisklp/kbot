APP := $(shell basename $(shell git remote get-url origin))
REGISTRY := mrfwdmail/kbot
VERSION=$(shell git describe --tags --abbrev=0)-$(shell git rev-parse --short HEAD)
TARGETOS=linux
TARGETARCH=amd64

format:
	gofmt -s -w ./

lint:
	golint

test:
	go test -v

get:
	go get

build: format get
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o kbot -ldflags "-X="github.com/denisklp/kbot/cmd.appVersion=${VERSION}

image:
	docker build . -t ${REGISTRY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH} --build-arg TARGETOS=${TARGETOS} --build-arg TARGETARCH=${TARGETARCH} 

push:
	docker push ${REGISTRY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH}

clean:
	rm -rf kbot
	docker rmi ${REGISTRY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH}

linux:
	$(MAKE) build TARGETOS=linux TARGETARCH=amd64


darwin:
	$(MAKE) build TARGETOS=darwin TARGETARCH=amd64


windows:
	$(MAKE) build TARGETOS=windows TARGETARCH=amd64


arm:
	$(MAKE) build TARGETOS=linux TARGETARCH=arm64
	