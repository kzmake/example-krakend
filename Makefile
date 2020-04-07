.PHONY: all fmt tools build clean fmt info help

GO               ?= go

SOURCES          := $(shell find . -name "*.go" -type f)
BUILD_FLAGS      := -a -ldflags "-w -extldflags -static" -tags "$(BUILD_TAGS)"

.DEFAULT_GOAL    := help

all: clean fmt build

tools: ## install tools
	@hash goimports > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		env GO111MODULE=off $(GO) get -u golang.org/x/tools/cmd/goimports; \
	fi
	@env GO111MODULE=off $(GO) get -u golang.org/x/tools/cmd/goimports

fmt: tools ## format codes
	goimports -l -w $(SOURCES)

build: ## build app
	@mkdir -p target
	$(GO) build $(BUILD_FLAGS) -o target/example-krakend github.com/kzmake/example-krakend

clean: ## clean up artifacts
	$(RM) -rf coverage/ target/
	$(GO) clean

help: ## show help
	@cat $(MAKEFILE_LIST) \
	| grep -e "^[a-zA-Z_/\-]*: *.*## *" \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' \
	| sed 's/\(.*\/.*\)/  \1/'
