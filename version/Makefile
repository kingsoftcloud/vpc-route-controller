# Project main package location (can be multiple ones).
CMD_DIR := ./cmd/manager

# Project output directory.
OUTPUT_DIR := ./output

TAG?=$(shell git describe --tags)
VERSION := latest

# Ksyun repository
BJKSYUNREPOSITORY:= hub.kce.ksyun.com/ksyun/vpc-route-controller

# ldflags
VERSION_PKG=newgit.op.ksyun.com/kce/vpc-route-controller/version
GIT_COMMIT=$(shell git rev-parse HEAD)
BUILD_DATE=$(shell date +%Y-%m-%dT%H:%M:%S%z)

CIPHER_KEY="8cca0smDmR478v8F"
AKSK_PKG=newgit.op.ksyun.com/kce/vpc-route-controller/pkg/aksk

ldflags="-s -w -X ${AKSK_PKG}.DefaultCipherKey=${CIPHER_KEY} -X $(VERSION_PKG).Version=$(TAG) -X $(VERSION_PKG).GitCommit=${GIT_COMMIT} -X ${VERSION_PKG}.BuildDate=${BUILD_DATE}"

all: compile build tag push

fmt:
	find ./ pkg cmd -type f -name "*.go" | xargs gofmt -l -w

compile:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=auto go build -o $(OUTPUT_DIR)/vpc-route-controller -ldflags $(ldflags) $(CMD_DIR)/main.go

build: 
	docker build -t vpc-route-controller:$(VERSION) -f Dockerfile .

tag:
	docker tag vpc-route-controller:$(VERSION) $(BJKSYUNREPOSITORY):$(VERSION)

push:
	docker push $(BJKSYUNREPOSITORY):$(VERSION)

.PHONY: clean
clean:
	rm -vrf ${OUTPUT_DIR}/

