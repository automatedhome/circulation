GO111MODULE=on
export GO111MODULE

IMAGE=automatedhome/circulation

.PHONY: build
build:
	go build -o circulation cmd/main.go

qemu-arm-static:
	./hooks/post_checkout

.PHONY: image
image: qemu-arm-static
	./hooks/pre_build
	docker build -t $(IMAGE) .
