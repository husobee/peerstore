PKGS := $(shell go list ./... | grep -v /vendor)
VERSION ?= latest
BINARY := peerstore
PLATFORMS := windows linux darwin
GOPATH_FIRST = $(firstword $(subst :, ,$1))
LINTER := $(call GOPATH_FIRST, $(GOPATH))/bin/golint

$(LINTER):
	go get -u github.com/golang/lint/golint

.PHONY: test
test:
	go test $(PKGS) -cover

.PHONY: lint
lint: $(LINTER)
	golint $(PKGS)

.PHONY: vet
vet:
	go vet $(PKGS)

os = $(word 1, $@)
.PHONY: $(PLATFORMS)
$(PLATFORMS):
	mkdir -p release
	GOOS=$(os) GOARCH=amd64 go build -o release/$(BINARY)_client-$(VERSION)-$(os)-amd64 cmd/peerstore/client/main.go
	GOOS=$(os) GOARCH=amd64 go build -o release/$(BINARY)_server-$(VERSION)-$(os)-amd64 cmd/peerstore/server/main.go

.PHONY: release
release: windows linux darwin
