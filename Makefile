.PHONY: test lint fmt

# nostos: **/*.go lang/itemtype_string.go
# 	go build github.com/wycleffsean/nostos/cmd/nostos
# Build the project (assuming your main package is in cmd/nostos)
bin/nostos: **/*.go lang/itemtype_string.go
	go build -o bin/nostos .

.PHONY: plan
plan: bin/nostos
	$< plan

# requires go install golang.org/x/tools/cmd/stringer
lang/itemtype_string.go: lang/lex.go
	go generate ./lang

test: lang/itemtype_string.go
	go test -v ./...

watch: lang/itemtype_string.go
	ls **/*.go | entr -c $(MAKE) test

# Run the linter (if you add one, e.g., golangci-lint)
lint:
	golangci-lint run

# this requires
# - go install golang.org/x/tools/dlv@latest
# - go install golang.org/x/tools/gdlv@latest
.PHONY: debug
debug:
	gdlv test lang

.PHONY: setup
setup:
	go install golang.org/x/tools/cmd/stringer
	go install github.com/spf13/cobra-cli@latest
	# go install golang.org/x/tools/dlv@latest
	# go install golang.org/x/tools/gdlv@latest

# Format the code
fmt:
	go fmt ./...
