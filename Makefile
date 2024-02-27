GOARCH = amd64
APPS ?= c2 carbon-proxy

DATE=$(shell date "+%Y%m%d_%H%M")
VERSION=v0.15.0
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
PWD=$(shell pwd)
# Symlink into GOPATH
BUILD_DIR=${PWD}
TARGET_DIR=${PWD}/target
PROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/

GOENV=GOPROXY=${PROXY} GOARCH=${GOARCH}

CURRENT_DIR=$(shell pwd)

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X github.com/alibaba/kube-sharding/pkg/ago-util/utils.date=${DATE} -X github.com/alibaba/kube-sharding/pkg/ago-util/utils.version=${VERSION}  -X github.com/alibaba/kube-sharding/pkg/ago-util/utils.commit=${COMMIT} -X github.com/alibaba/kube-sharding/pkg/ago-util/utils.branch=${BRANCH}"

# Build the project
all: clean vet ut linux darwin
aone: clean vet linux

link:
	if [ "$${BUILD_DIR}" != "$${CURRENT_DIR}" ]; then \
	    echo "ERROR: Clone project to [${BUILD_DIR}] to build"; \
	fi

linux:
	for i in $(APPS); do \
		echo ${PWD}/cmd/$$i ;\
		cd ${PWD}/cmd/$$i; \
		${GOENV} GOOS=linux go build ${LDFLAGS} -o ${PWD}/target/$$i/$$i-linux-${GOARCH}; \
	done
	cp -rf ${PWD}/etc/ ${PWD}/target/etc
	cp -rf ${PWD}/assets/ ${PWD}/target/assets
	${GOENV} GOOS=linux go build ${LDFLAGS} -o ${PWD}/target/c2/memorize-linux ${BUILD_DIR}/cmd/c2/memorize/memorize.go

darwin:
	for i in $(APPS); do \
		echo ${PWD}/cmd/$$i ;\
		cd ${PWD}/cmd/$$i; \
		${GOENV} GOOS=darwin go build ${LDFLAGS} -o ${PWD}/target/$$i/$$i-darwin-${GOARCH}; \
	done
	cp -rf ${PWD}/etc/ ${PWD}/target/etc
	cp -rf ${PWD}/assets/ ${PWD}/target/assets
	${GOENV} GOOS=darwin go build ${LDFLAGS} -o ${PWD}/target/c2/memorize-darwin ${BUILD_DIR}/cmd/c2/memorize/memorize.go

ut:
	${GOENV} go test -v ./...

vet:
	${GOENV} go vet ./... 

GO_SUBPKGS = $(shell go list ./... | grep -v /vendor/ | grep -v /vendor/ )

lint:
	@echo "golint"
	@for f in $(GO_SUBPKGS) ; do golint $$f ; done
	@echo ""

fmt:
	cd ${BUILD_DIR}; \
	go fmt $$(go list ./... | grep -v /vendor/) ; \

clean:
	-rm -rf ${TARGET_DIR}

