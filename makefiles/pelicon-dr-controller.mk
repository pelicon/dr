PELICON_DR_NAME = pelicon-dr-controller
PELICON_DR_IMAGE_DIR = ${PROJECT_SOURCE_CODE_DIR}/build
PELICON_DR_BUILD_BIN = ${BINS_DIR}/${PELICON_DR_NAME}-run
PELICON_DR_BUILD_MAIN = ${CMDS_DIR}/manager/main.go

PELICON_DR_RELEASE_IMAGE_NAME = ${IMAGE_REGISTRY}/${PELICON_DR_NAME}

.PHONY: pelicon_dr
pelicon_dr:
	GOARCH=amd64 ${BUILD_ENVS} ${BUILD_CMD} ${BUILD_OPTIONS} -o ${PELICON_DR_BUILD_BIN} ${PELICON_DR_BUILD_MAIN}

.PHONY: pelicon_dr_arm64
pelicon_dr_arm64:
	GOARCH=arm64 ${BUILD_ENVS} ${BUILD_CMD} ${BUILD_OPTIONS} -o ${PELICON_DR_BUILD_BIN} ${PELICON_DR_BUILD_MAIN}

.PHONY: pelicon_dr_release
pelicon_dr_release:
	# build for amd64 version
	${DOCKER_MAKE_CMD} make pelicon_dr
	${DOCKER_BUILDX_CMD_AMD64} -t ${PELICON_DR_RELEASE_IMAGE_NAME}:${RELEASE_TAG}-amd64 -f ${PELICON_DR_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	# build for arm64 version
	${DOCKER_MAKE_CMD} make pelicon_dr_arm64
	${DOCKER_BUILDX_CMD_ARM64} -t ${PELICON_DR_RELEASE_IMAGE_NAME}:${RELEASE_TAG}-arm64 -f ${PELICON_DR_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	# push to a public registry
	${MUILT_ARCH_PUSH_CMD} ${PELICON_DR_RELEASE_IMAGE_NAME}:${RELEASE_TAG}

