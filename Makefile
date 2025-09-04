BINARY_NAME=helm-template-service

.PHONY: build container run

build:
	go build -o $(BINARY_NAME) .

container:
	podman build -t $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)