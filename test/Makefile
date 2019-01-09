GOFMT=gofmt
GC=go build
VERSION := $(shell git describe --abbrev=4 --always --tags)
BUILD_PAR = -ldflags "-X github.com/oniio/oniChannel/channel.Version=$(VERSION)"

SRC_FILES = $(shell git ls-files | grep -e .go$ | grep -v _test.go)

channel: $(SRC_FILES)
	$(GC)  $(BUILD_PAR) -o channel main.go

all: channel


format:
	$(GOFMT) -w main.go

clean:
	rm -rf *.8 *.o *.out *.6 *exe
	rm -rf channel channel-*
