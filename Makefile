include Makefile.common

GOCMD?=go

FIND_MOD_ARGS=-type f -name "go.mod"  -not -path "./ta/*"
TO_MOD_DIR=dirname {} \; | sort | egrep  '^./'

ALL_MODS := $(shell find . $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR))

.PHONY := tgz
tgz: build
	tar --format ustar -C ta -czvf splunk-connect-for-otlp.tgz splunk-connect-for-otlp

splunk-connect-for-otlp.tgz: tgz

ta/splunk-connect-for-otlp/linux_x86_64/bin/splunk-connect-for-otlp: $(shell find . -name "*.go" -type f -not -path "./ta/*")
	mkdir -p ../../ta/splunk-connect-for-otlp/linux_x86_64/bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -C cmd/splunk-connect-for-otlp -trimpath -o ../../ta/splunk-connect-for-otlp/linux_x86_64/bin/splunk-connect-for-otlp .

ta/splunk-connect-for-otlp/windows_x86_64/bin/splunk-connect-for-otlp: $(shell find . -name "*.go" -type f -not -path "./ta/*")
	mkdir -p ../../ta/splunk-connect-for-otlp/windows_x86_64/bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -C cmd/splunk-connect-for-otlp -trimpath -o ../../ta/splunk-connect-for-otlp/windows_x86_64/bin/splunk-connect-for-otlp .

.PHONY := build
build: ta/splunk-connect-for-otlp/linux_x86_64/bin/splunk-connect-for-otlp ta/splunk-connect-for-otlp/windows_x86_64/bin/splunk-connect-for-otlp

.PHONY := splunk
splunk: splunk-connect-for-otlp.tgz
	docker run --rm -it -v $(PWD)/splunk-connect-for-otlp.tgz:/tmp/splunk-connect-for-otlp.tgz \
		--platform linux/amd64 \
		-e "SPLUNK_PASSWORD=changeme" \
		-e "SPLUNK_APPS_URL=file:///tmp/splunk-connect-for-otlp.tgz" \
		-e "SPLUNK_GENERAL_TERMS=--accept-sgt-current-at-splunk-com" \
		-e "SPLUNK_START_ARGS=--accept-license" \
		-p 4317:4317 \
		-p 4318:4318 \
		-p 8000:8000 \
		splunk/splunk:10.0

.PHONY: install-tools
install-tools:
	cd ./internal/tools && go install github.com/client9/misspell/cmd/misspell
	cd ./internal/tools && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	cd ./internal/tools && go install github.com/google/addlicense
	cd ./internal/tools && go install golang.org/x/tools/cmd/goimports
	cd ./internal/tools && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	cd ./internal/tools && go install mvdan.cc/gofumpt

# Define a delegation target for each module
.PHONY: $(ALL_MODS)
$(ALL_MODS):
	@echo "Running target '$(TARGET)' in module '$@'"
	$(MAKE) --no-print-directory -C $@ $(TARGET)

# Triggers each module's delegation target
.PHONY: for-all-target
for-all-target: $(ALL_MODS)

.PHONY: tidy-all
tidy-all:
	$(MAKE) for-all-target TARGET="tidy"
	$(MAKE) tidy

.PHONY: fmt-all
fmt-all:
	$(MAKE) for-all-target TARGET="fmt"
	$(MAKE) fmt

.PHONY: lint-all
lint-all:
	$(MAKE) for-all-target TARGET="lint"
	$(MAKE) lint

.PHONY: test-all
test-all:
	$(MAKE) for-all-target TARGET="test"
	$(MAKE) test

.PHONY: benchmark-all
benchmark-all:
	$(MAKE) for-all-target TARGET="benchmark"

ifeq ($(COVER_TESTING),true)
# These targets are expensive to build, so only build if explicitly requested

.PHONY: gotest-with-codecov
gotest-with-codecov:
	@$(MAKE) for-all-target TARGET="test-with-codecov"
	@$(MAKE) test-with-codecov
	$(GOCMD) tool covdata textfmt -i=./coverage -o ./coverage.txt

endif
