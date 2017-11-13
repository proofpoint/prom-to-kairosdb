NS ?= rajatjindal
VERSION ?= latest

REPO = prom-to-kairosdb
NAME = prom-to-kairosdb
INSTANCE = default

.PHONY: build push shell run start stop rm release vendor

default: fmt vet test build

build: vendor
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/prom-to-kairosdb .

docker:
	docker build -t $(NS)/$(REPO):$(VERSION) .

push:
	docker push $(NS)/$(REPO):$(VERSION)

rm:
	docker rm $(NAME)-$(INSTANCE)

release: docker
	make push -e VERSION=$(VERSION)

test:
	go test ./... -cover

fmt:
	go fmt ./...

vet:
	go vet ./...

vendor:
	go get -u github.com/Masterminds/glide
	glide install

