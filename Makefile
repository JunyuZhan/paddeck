# PadDeck — common tasks
#
#   make build     compile ./paddeck
#   make run       run from a checkout (auto-discovers config.yaml)
#   make check     compile + probe every configured target
#   make test      unit tests
#   make theme     regenerate theme CSS from web/style.css
#   make docker    build and start the container
#   make verify    everything CI runs

BINARY  ?= paddeck
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOFLAGS ?=
LDFLAGS := -s -w -X main.version=$(VERSION)
GOPROXY ?= https://goproxy.cn,direct

.PHONY: help build run check test vet fmt theme verify docker clean

help:
	@sed -n '2,12p' Makefile | sed 's/^# \{0,1\}//'

build:
	CGO_ENABLED=0 GOPROXY=$(GOPROXY) go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/paddeck

# cross-compile for a linux amd64 host
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=$(GOPROXY) go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY)-linux ./cmd/paddeck

run:
	GOPROXY=$(GOPROXY) go run ./cmd/paddeck

check: build
	./$(BINARY) -check

test:
	GOPROXY=$(GOPROXY) go test ./...

vet:
	GOPROXY=$(GOPROXY) go vet ./...
	gofmt -l . | (! grep .) || (echo "run: gofmt -w ." && exit 1)

theme:
	python3 tools/themes.py

docker:
	docker compose up -d --build

verify: vet test theme
	git diff --exit-code -- web/theme-*.css web/themes.js
	gh_secret_scan() { ! git ls-files | grep -qE '^(config\.yaml|config\.local\.yaml|keys/|data/)'; }; gh_secret_scan
	CGO_ENABLED=0 GOPROXY=$(GOPROXY) go build -trimpath -o /tmp/paddeck-ci ./cmd/paddeck

clean:
	rm -f $(BINARY) $(BINARY)-linux /tmp/paddeck-ci
