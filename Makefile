BINARY := c3x
PKG := github.com/c3xdev/c3x/cmd/c3x
VERSION := $(shell scripts/get_version.sh HEAD $(NO_DIRTY))
LD_FLAGS := -ldflags="-s -X 'github.com/c3xdev/c3x/internal/version.Version=$(VERSION)'"
BUILD_FLAGS := $(LD_FLAGS) -v

DEV_ENV := dev
ifdef C3X_ENV
	DEV_ENV := $(C3X_ENV)
endif

.PHONY: deps run build windows linux darwin build_all install release clean test fmt lint

deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go mod download

run:
	env C3X_ENV=$(DEV_ENV) go run $(LD_FLAGS) $(PKG) $(ARGS)

jsonschema:
	go run ./cmd/jsonschema/main.go --out-file ./schema/c3x.schema.json
	go run ./cmd/jsonschema/main.go --out-file ./schema/config.schema.json --schema config

tagschema:
	cd internal/providers/terraform/provider_schemas && \
	terraform init && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/aws".resource_schemas | to_entries | map(select(.value.block.attributes.tags)) | from_entries | with_entries(.value = true)' > aws.tags.json && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/aws".resource_schemas | to_entries | map(select(.value.block.attributes.tags_all)) | from_entries | with_entries(.value = true)' > aws.tags_all.json && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/aws".resource_schemas | to_entries | map(select(.value.block.block_types.tag)) | from_entries | with_entries(.value = true)' > aws.tag_block.json && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/azurerm".resource_schemas | to_entries | map(select(.value.block.attributes.tags)) | from_entries | with_entries(.value = true)' > azurerm.tags.json && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/google".resource_schemas | to_entries | map(select(.value.block.attributes.labels)) | from_entries | with_entries(.value = true)' > google.labels.json && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/google".resource_schemas | to_entries | map(select(.value.block.attributes.user_labels)) | from_entries | with_entries(.value = true)' > google.user_labels.json && \
	terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/hashicorp/google".resource_schemas | to_entries | map(select(.value.block.block_types.settings.block.attributes.user_labels)) | from_entries | with_entries(.value = true)' > google.settings_user_labels.json

build:
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY) $(PKG)

windows:
	env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY).exe $(PKG)
	env GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY)-arm64.exe $(PKG)

linux:
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY)-linux-amd64 $(PKG)
	env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY)-linux-arm64 $(PKG)

darwin:
	env GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY)-darwin-amd64 $(PKG)
	env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/$(BINARY)-darwin-arm64 $(PKG)

build_all: build windows linux darwin

install:
	CGO_ENABLED=0 go install $(BUILD_FLAGS) $(PKG)

release: build_all
	cd build; tar -czf $(BINARY)-windows-amd64.tar.gz $(BINARY).exe; shasum -a 256 $(BINARY)-windows-amd64.tar.gz > $(BINARY)-windows-amd64.tar.gz.sha256
	cd build; zip -r $(BINARY)-windows-amd64.zip $(BINARY).exe; shasum -a 256 $(BINARY)-windows-amd64.zip > $(BINARY)-windows-amd64.zip.sha256
	cd build; tar -czf $(BINARY)-windows-arm64.tar.gz $(BINARY)-arm64.exe; shasum -a 256 $(BINARY)-windows-arm64.tar.gz > $(BINARY)-windows-arm64.tar.gz.sha256
	cd build; mv $(BINARY)-arm64.exe $(BINARY).exe; zip -r $(BINARY)-windows-arm64.zip $(BINARY).exe; shasum -a 256 $(BINARY)-windows-arm64.zip > $(BINARY)-windows-arm64.zip.sha256
	cd build; tar -czf $(BINARY)-linux-amd64.tar.gz $(BINARY)-linux-amd64; shasum -a 256 $(BINARY)-linux-amd64.tar.gz > $(BINARY)-linux-amd64.tar.gz.sha256
	cd build; tar -czf $(BINARY)-linux-arm64.tar.gz $(BINARY)-linux-arm64; shasum -a 256 $(BINARY)-linux-arm64.tar.gz > $(BINARY)-linux-arm64.tar.gz.sha256
	cd build; tar -czf $(BINARY)-darwin-amd64.tar.gz $(BINARY)-darwin-amd64; shasum -a 256 $(BINARY)-darwin-amd64.tar.gz > $(BINARY)-darwin-amd64.tar.gz.sha256
	cd build; tar -czf $(BINARY)-darwin-arm64.tar.gz $(BINARY)-darwin-arm64; shasum -a 256 $(BINARY)-darwin-arm64.tar.gz > $(BINARY)-darwin-arm64.tar.gz.sha256

clean:
	go clean
	rm -rf build/$(BINARY)*

# Run only short unit tests
test:
	C3X_LOG_LEVEL=warn go test -short $(LD_FLAGS) ./... $(or $(ARGS), -cover)

# Run only short unit tests with parallelism disabled and verbosity true.
# This is useful for pinpointing a hanging test.
test_verbose:
	C3X_LOG_LEVEL=warn go test -p 1 -v -short $(LD_FLAGS) ./... $(or $(ARGS), -cover)

# Run all tests
test_all:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./... $(or $(ARGS), -cover)

# Run unit tests and shared integration tests (excludes cloud provider golden file tests)
test_shared_int:
	C3X_LOG_LEVEL=error go test -timeout 30m $(LD_FLAGS) \
		$(shell go list ./... | grep -v ./internal/providers/terraform/aws | grep -v ./internal/providers/terraform/google | grep -v ./internal/providers/terraform/azure | grep -v ./internal/cloud/terraform/aws | grep -v ./internal/cloud/terraform/azure | grep -v ./internal/cloud/terraform/google | grep -v ./cmd/c3x) \
		$(ARGS)

# Run golden file tests that compare against the pricing API (run after test_shared_int)
test_golden:
	C3X_LOG_LEVEL=error go test -timeout 60m $(LD_FLAGS) \
		./cmd/c3x \
		./internal/cloud/terraform/aws \
		./internal/cloud/terraform/azure \
		./internal/cloud/terraform/google \
		$(ARGS)

test_cmd:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./cmd/c3x $(or $(ARGS), -cover)

test_update_cmd:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./cmd/c3x $(or $(ARGS), -update -cover)

test_usage:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/usage $(or $(ARGS), -cover)

test_update_usage:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/usage $(or $(ARGS), -update -cover)

# Run AWS resource tests
test_aws:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/terraform/aws $(or $(ARGS), -cover)

# Run Google resource tests
test_google:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/terraform/google $(or $(ARGS), -cover)

# Run Azure resource tests
test_azure:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/terraform/azure $(or $(ARGS), -cover)

# Update AWS golden files tests
test_update:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/... $(or $(ARGS), -update -cover)

# Update golden files tests
test_update_aws:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/terraform/aws $(or $(ARGS), -update -cover)

# Update Google golden files tests
test_update_google:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/terraform/google $(or $(ARGS), -update -cover)

# Update Azure golden files tests
test_update_azure:
	C3X_LOG_LEVEL=warn go test -timeout 30m $(LD_FLAGS) ./internal/providers/terraform/azure $(or $(ARGS), -update -cover)

fmt:
	go fmt ./...
	@find . -path '*/.*' -prune -o -name '*_with_error.tf' -prune -o -name '*.tf' -type f -print0 | xargs -0 -L1 bash -c '! test "$$(tail -c 1 "$$0")" || (echo >> "$$0" && echo "Terminating newline added to $$0")'
	find . -path '*/.*' -prune -o -name '*_with_error.tf' -prune -o -name '*.tf' -exec terraform fmt {} +;

tf_fmt_check:
	@echo "Checking Terraform formatting..."
	@if ! find . -path '*/.*' -prune -o -name '*_with_error.tf' -prune -o -name '*.tf' -exec terraform fmt -check=true -write=false {} +; then \
		echo "Terraform files not formatted. Run 'make fmt' to fix, or rename the file to '..._with_error.tf' to ignore."; \
		exit 1; \
	fi
	@echo "Checking that .tf files end with newline..."
	@if ! find . -path '*/.*' -prune -o -name '*_with_error.tf' -prune -o -name '*.tf' -type f -print0 | xargs -0 -L1 bash -c '! test "$$(tail -c 1 "$$0")" || (echo "$$0" && false)'; then \
		echo "Terraform files missing terminating newline. Run 'make fmt' to fix, or rename the file to '..._with_error.tf' to ignore."; \
        exit 1; \
    fi

lint:
	golangci-lint run

parser_benchmarks:
	go test -bench=. -benchmem -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out github.com/c3xdev/c3x/internal/iac/hcl
