include Makefile.common

GOCMD?=go

FIND_MOD_ARGS=-type f -name "go.mod"  -not -path "./ta/*"
TO_MOD_DIR=dirname {} \; | sort | egrep  '^./'

ALL_MODS := $(shell find . $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR))

.PHONY := tgz
tgz: build
	tar --format ustar -C ta -czvf splunk-connect-for-otlp.tgz splunk-connect-for-otlp

splunk-connect-for-otlp.tgz: tgz

ta/splunk-connect-for-otlp/linux_x86_64/bin/splunk-connect-for-otlp: $(shell find  **/*.go -type f)
	mkdir -p ../../ta/splunk-connect-for-otlp/linux_x86_64/bin
	GOOS=linux GOARCH=amd64 go build -C cmd/splunk-connect-for-otlp -trimpath -o ../../ta/splunk-connect-for-otlp/linux_x86_64/bin/splunk-connect-for-otlp .

ta/splunk-connect-for-otlp/windows_x86_64/bin/splunk-connect-for-otlp: $(shell find  **/*.go -type f)
	mkdir -p ../../ta/splunk-connect-for-otlp/windows_x86_64/bin
	GOOS=windows GOARCH=amd64 go build -C cmd/splunk-connect-for-otlp -trimpath -o ../../ta/splunk-connect-for-otlp/windows_x86_64/bin/splunk-connect-for-otlp .

.PHONY := build
build: ta/splunk-connect-for-otlp/linux_x86_64/bin/splunk-connect-for-otlp ta/splunk-connect-for-otlp/windows_x86_64/bin/splunk-connect-for-otlp

.PHONY := splunk
splunk: splunk-connect-for-otlp.tgz
	docker run --rm -it -v $(PWD)/splunk-connect-for-otlp.tgz:/tmp/splunk-connect-for-otlp.tgz \
		-e "SPLUNK_PASSWORD=changeme" \
		-e "SPLUNK_APPS_URL=file:///tmp/splunk-connect-for-otlp.tgz" \
		-e "SPLUNK_START_ARGS=--accept-license" \
		-e "SPLUNK_HEC_TOKEN=000000-0000-00000-0000000000" \
		-p 4317:4317 \
		-p 4318:4318 \
		-p 8000:8000 \
		splunk/splunk:9.3

# Define a delegation target for each module
.PHONY: $(ALL_MODS)
$(ALL_MODS):
	@echo "Running target '$(TARGET)' in module '$@'"
	$(MAKE) --no-print-directory -C $@ $(TARGET)

# Triggers each module's delegation target
.PHONY: for-all-target
for-all-target: $(ALL_MODS)

.PHONY: test-all
test-all:
	$(MAKE) for-all-target TARGET="test"
	$(MAKE) test

.PHONY: benchmark-all
benchmark-all:
	$(MAKE) for-all-target TARGET="benchmark"

.PHONY: gotidy
gotidy:
	@for mod in $$(find . -name go.mod | xargs dirname); do \
		echo "Tidying $$mod"; \
		(cd $$mod && rm -rf go.sum && $(GOCMD) mod tidy -compat=1.24.0 && $(GOCMD) get toolchain@none) || exit $?; \
	done

ifeq ($(COVER_TESTING),true)
# These targets are expensive to build, so only build if explicitly requested

.PHONY: gotest-with-codecov
gotest-with-codecov:
	@$(MAKE) for-all-target TARGET="test-with-codecov"
	@$(MAKE) test-with-codecov
	$(GOCMD) tool covdata textfmt -i=./coverage -o ./coverage.txt

endif
