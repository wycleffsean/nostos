nostos: **/*.go lang/itemtype_string.go
	go build github.com/wycleffsean/nostos/cmd/nostos

# requires go install golang.org/x/tools/cmd/stringer
lang/itemtype_string.go:
	go generate ./lang

.PHONY: test
test:
	go generate ./lang
	go test github.com/wycleffsean/nostos/lang

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
