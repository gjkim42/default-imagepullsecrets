VERSION?=$(shell if [ -d .git ]; then git describe --tags --dirty; else echo "unknown"; fi)
REGISTRY?=gjkim42
BASEIMAGE=gcr.io/distroless/static-debian11
GO_VERSION?=1.17
OS_CODENAME?=bullseye
OUTPUT_DIR?=_output

.PHONY: build
build:
	go build -o ${OUTPUT_DIR}/default-imagepullsecrets ./cmd/default-imagepullsecrets

.PHONY: verify
verify:
	hack/verify.sh

.PHONY: update
update:
	hack/update.sh

.PHONY: test
test:
	hack/make-rules/test.sh $(WHAT)

.PHONY: image
image:
	docker build \
		-t ${REGISTRY}/default-imagepullsecrets:${VERSION} \
		--build-arg GO_VERSION=${GO_VERSION} \
		--build-arg OS_CODENAME=${OS_CODENAME} \
		--build-arg BASEIMAGE=${BASEIMAGE} \
		--build-arg OUTPUT_DIR=${OUTPUT_DIR} \
		.

.PHONY: push
push:
	docker push ${REGISTRY}/default-imagepullsecrets:${VERSION}
