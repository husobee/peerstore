PKGS := $(shell go list ./... | grep -v /vendor)

.PHONY: test
test:
	go test $(PKGS) -cover

.PHONY: lint
lint:
	go get github.com/golang/lint/golint
	golint $(PKGS)

.PHONY: vet
vet:
	go vet $(PKGS)

SRV_BINARY := peerstore_server
CLI_BINARY := peerstore_client

.PHONY: windows
windows:
	mkdir -p release
	GOOS=windows GOARCH=amd64 go build -o release/$(CLI_BINARY)-$(VERSION)-windows-amd64 cmd/peerstore/client/main.go
	GOOS=windows GOARCH=amd64 go build -o release/$(SRV_BINARY)-$(VERSION)-windows-amd64 cmd/peerstore/server/main.go

.PHONY: linux
linux:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -o release/$(CLI_BINARY)-$(VERSION)-linux-amd64 cmd/peerstore/client/main.go
	GOOS=linux GOARCH=amd64 go build -o release/$(SRV_BINARY)-$(VERSION)-linux-amd64 cmd/peerstore/server/main.go

.PHONY: darwin
darwin:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 go build -o release/$(CLI_BINARY)-$(VERSION)-darwin-amd64 cmd/peerstore/client/main.go
	GOOS=darwin GOARCH=amd64 go build -o release/$(SRV_BINARY)-$(VERSION)-darwin-amd64 cmd/peerstore/server/main.go

.PHONY: release
release: windows linux darwin
