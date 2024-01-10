# Project main package location (can be multiple ones).
CMD_DIR := ./cmd/manager

# Project output directory.
OUTPUT_DIR := ./output

VERSION ?= v1.0.0

# Ksyun repository
BJKSYUNREPOSITORY ?= hub.kce.ksyun.com/ksyun/vpc-route-controller

# ldflags
VERSION_PKG=ezone.ksyun.com/ezone/kce/vpc-route-controller/version
GIT_COMMIT=$(shell git rev-parse HEAD)
BUILD_DATE=$(shell date +%Y-%m-%dT%H:%M:%S%z)

CIPHER_KEY=$(shell echo "yourcipherkey")
KSYUN_PKG=ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun
ALARM_PKG=ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/alarm
AK_FOR_ALARM=$(shell echo "yourakforalarmopenapi")
SK_FOR_ALARM=$(shell echo "yourskforalarmopenapi")

ldflags="-s -w -X ${KSYUN_PKG}.DefaultCipherKey=${CIPHER_KEY} -X ${ALARM_PKG}.AKForAlarm=${AK_FOR_ALARM} -X ${ALARM_PKG}.SKForAlarm=${SK_FOR_ALARM} -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).GitCommit=${GIT_COMMIT} -X ${VERSION_PKG}.BuildDate=${BUILD_DATE}"

all: compile build tag push

fmt:
	find ./ pkg cmd -type f -name "*.go" | xargs gofmt -l -w

compile:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) GO111MODULE=auto go build -o $(OUTPUT_DIR)/vpc-route-controller -ldflags $(ldflags) $(CMD_DIR)/main.go

build:
	docker build -t vpc-route-controller:$(VERSION)-$(ARCH) -f ./dockerfile/Dockerfile.$(ARCH) .

tag:
	docker tag vpc-route-controller:$(VERSION)-$(ARCH) $(BJKSYUNREPOSITORY):$(VERSION)-$(ARCH)

push:
	docker push $(BJKSYUNREPOSITORY):$(VERSION)-$(ARCH)

annotation-all: annotation-compile annotation-build annotation-tag annotation-push

annotation-compile:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) GO111MODULE=auto go build -o $(OUTPUT_DIR)/annotation ./cmd/annotation/main.go

annotation-build:
	docker build -t  annotation:$(VERSION)-$(ARCH) -f ./dockerfile/annotation.Dockerfile.$(ARCH) .

annotation-tag:
	docker tag annotation:$(VERSION)-$(ARCH) $(BJKSYUNREPOSITORY)/annotation:$(VERSION)-$(ARCH)

annotation-push:
	docker push $(BJKSYUNREPOSITORY)/annotation:$(VERSION)-$(ARCH)

.PHONY: clean
clean:
	rm -vrf ${OUTPUT_DIR}/

