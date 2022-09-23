IMAGE_REGISTRY ?= ghcr.io/pelicon

GO_VERSION = $(shell go version)
BUILD_TIME = ${shell date +%Y-%m-%dT%H:%M:%SZ}
BUILD_VERSION = ${shell git rev-parse --short "HEAD^{commit}" 2>/dev/null}
BUILD_ENVS = CGO_ENABLED=0 GOOS=linux
BUILD_FLAGS = -X 'main.BUILDVERSION=${BUILD_VERSION}' -X 'main.BUILDTIME=${BUILD_TIME}' -X 'main.GOVERSION=${GO_VERSION}'
BUILD_OPTIONS = -a -mod vendor -installsuffix cgo -ldflags "${BUILD_FLAGS}"

PROJECT_SOURCE_CODE_DIR=$(CURDIR)
BINS_DIR = ${PROJECT_SOURCE_CODE_DIR}/_build
CMDS_DIR = ${PROJECT_SOURCE_CODE_DIR}/cmd
IMAGES_DIR = ${PROJECT_SOURCE_CODE_DIR}/images

BUILD_CMD = go build
OPERATOR_CMD = operator-sdk
RUN_CMD = go run
K8S_CMD = kubectl

BUILDER_NAME = ${IMAGE_REGISTRY}/dr/builder
BUILDER_TAG = v0.1
BUILDER_MOUNT_SRC_DIR = ${PROJECT_SOURCE_CODE_DIR}/../
BUILDER_MOUNT_DST_DIR = /go/src/github.com/pelicon
BUILDER_WORKDIR = /go/src/github.com/pelicon/dr

DOCKER_SOCK_PATH=/var/run/docker.sock
DOCKER_MAKE_CMD = docker run --rm -v ${BUILDER_MOUNT_SRC_DIR}:${BUILDER_MOUNT_DST_DIR} -v ${DOCKER_SOCK_PATH}:${DOCKER_SOCK_PATH} -w ${BUILDER_WORKDIR} -i ${BUILDER_NAME}:${BUILDER_TAG}
DOCKER_BUILDX_CMD_AMD64 = DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform=linux/amd64 -o type=docker
DOCKER_BUILDX_CMD_ARM64 = DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform=linux/arm64 -o type=docker
MUILT_ARCH_PUSH_CMD = ${PROJECT_SOURCE_CODE_DIR}/docker-push-with-multi-arch.sh

IMAGE_TAG = v99.9.9
RELEASE_TAG ?= $(shell tagged="$$(git describe --tags --match='v*' --abbrev=0 2> /dev/null)"; if [ "$$tagged" ] && [ "$$(git rev-list -n1 HEAD)" = "$$(git rev-list -n1 $$tagged)" ]; then echo $$tagged; fi)

.PHONY: builder
builder:
	docker build -t ${BUILDER_NAME}:${BUILDER_TAG} -f images/builder/Dockerfile .
	docker push ${BUILDER_NAME}:${BUILDER_TAG}

.PHONY: debug
debug:
	${DOCKER_MAKE_CMD} ash

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: gen_code
gen_code:	
	${OPERATOR_CMD} generate k8s
	${OPERATOR_CMD} generate crds

.PHONY: clean
clean:
	go clean -r -x
	rm -rf ${BINS_DIR}
	docker container prune -f
	docker rmi -f $(shell docker images -f dangling=true -qa)
	
.PHONY: gen_client
gen_client:
	${DOCKER_MAKE_CMD} /code-generator/generate-groups.sh all github.com/pelicon/dr/pkg/apis/client github.com/pelicon/dr/pkg/apis "dr:v1alpha1" --go-header-file /code-generator/boilerplate.go.txt

.PHONY: release
release: pelicon_dr_release

unit-test:
	bash hack/unit_test.sh

include ./makefiles/pelicon-dr-controller.mk

