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

vuln:
	govulncheck $(GO_PACKAGES)

quality: fmt-check tidy-check vet lint race build

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

.PHONY: build fmt fmt-check tidy-check vet lint test race cover vuln quality release-check install benchmark docker-build docker-push docker
