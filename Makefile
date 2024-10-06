nostos: **/*.go
	go build github.com/wycleffsean/nostos/cmd/nostos

.PHONY: test
test:
	go test github.com/wycleffsean/nostos/lang

# this requires
# - go install golang.org/x/tools/dlv@latest
# - go install golang.org/x/tools/gdlv@latest
.PHONY: debug
debug:
	gdlv test lang
