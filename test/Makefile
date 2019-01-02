GOFMT=gofmt
GC=go build
VERSION := $(shell git describe --abbrev=4 --always --tags)
#BUILD_NIMBUS_PAR = -ldflags "-X github.com/oniio/dsp-go-sdk/common/config.VERSION=$(VERSION)"
BUILD_NIMBUS_PAR = --tags "json1"

SRC_FILES = $(shell git ls-files | grep -e .go$ | grep -v _test.go)

nimbus: $(SRC_FILES)
	$(GC)  $(BUILD_NIMBUS_PAR) -o nimbus main.go

all: nimbus

do-cross: w-nimbus l-nimbus d-nimbus

w-nimbus:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GC) $(BUILD_NIMBUS_PAR) -o do-windows-amd64.exe main.go

l-nimbus:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GC) $(BUILD_NIMBUS_PAR) -o do-linux-amd64 main.go

d-nimbus:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GC) $(BUILD_NIMBUS_PAR) -o do-darwin-amd64 main.go

format:
	$(GOFMT) -w main.go

clean:
	rm -rf *.8 *.o *.out *.6 *exe
	rm -rf nimbus nimbus-*
