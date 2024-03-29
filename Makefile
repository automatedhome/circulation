APP=circulation
IMAGE=automatedhome/$(APP)

.PHONY: build
build:
	go build -o $(APP) cmd/main.go

qemu-arm-static:
	./hooks/post_checkout

.PHONY: image
image: qemu-arm-static
	./hooks/pre_build
	docker build -t $(IMAGE) .
