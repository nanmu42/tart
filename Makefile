VERSION := $(shell git describe --tags --dirty)
REVISION := $(shell git rev-parse --short HEAD)

.PHONY: config clean dir all

all: clean tart

dir:
	mkdir -p bin

clean:
	rm -rf bin

tart: tart.bin

%.bin: dir
	cd cmd/$* && \
	go build -trimpath -ldflags "-s -w -X github.com/nanmu42/tart/version.Tag=$(VERSION) -X github.com/nanmu42/tart/version.Revision=$(REVISION)" && \
	cp $* $(PWD)/bin/$*