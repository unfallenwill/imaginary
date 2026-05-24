OK_COLOR=\033[32;01m
NO_COLOR=\033[0m
GO_PACKAGES=./...
GO_FILES=$$(find . -name '*.go' -not -path './vendor/*')

build:
	@echo "$(OK_COLOR)==> Compiling binary$(NO_COLOR)"
	go build -o bin/imaginary ./cmd/imaginary

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_FILES))"

tidy-check:
	go mod tidy
	git diff --exit-code go.mod go.sum

vet:
	go vet $(GO_PACKAGES)

lint:
	golangci-lint run

race:
	go test -race -count=1 $(GO_PACKAGES)

test:
	go test $(GO_PACKAGES)

cover:
	go test -coverprofile=coverage.out $(GO_PACKAGES)
	go tool cover -func=coverage.out

COVER_THRESHOLD=50

cover-check:
	@echo "$(OK_COLOR)==> Checking test coverage (threshold: $(COVER_THRESHOLD)%)$(NO_COLOR)"
	@go test -coverprofile=coverage.out $(GO_PACKAGES) 2>&1 | grep -E "^(ok|FAIL|---)" > /dev/null; \
	if [ ! -f coverage.out ]; then echo "Error: coverage.out not generated"; exit 1; fi; \
	COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $${COVERAGE}%"; \
	if [ "$$(echo "$$COVERAGE < $(COVER_THRESHOLD)" | bc -l)" = "1" ]; then \
		echo "Coverage $${COVERAGE}% is below threshold $(COVER_THRESHOLD)%"; \
		exit 1; \
	fi; \
	echo "$(OK_COLOR)Coverage check passed$(NO_COLOR)"

vuln:
	govulncheck $(GO_PACKAGES)

arch:
	go-arch-lint check

quality: fmt-check tidy-check vet lint arch race build

release-check: quality vuln docker-build

install:
	go install ./cmd/imaginary

benchmark: build
	bash benchmark.sh

docker-build:
	@echo "$(OK_COLOR)==> Building Docker image$(NO_COLOR)"
	docker build --no-cache=true --build-arg IMAGINARY_VERSION=$(VERSION) -t h2non/imaginary:$(VERSION) .

docker-push:
	@echo "$(OK_COLOR)==> Pushing Docker image v$(VERSION) $(NO_COLOR)"
	docker push h2non/imaginary:$(VERSION)

docker: docker-build docker-push

.PHONY: build fmt fmt-check tidy-check vet lint arch test race cover cover-check vuln quality release-check install benchmark docker-build docker-push docker
