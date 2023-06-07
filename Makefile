PROJECT = lipo-app

# # Paths ..........................
DESTDIR     =
PREFIX      = /usr/local

MAKEFILE_DIR ?= $(realpath $(dir $(lastword $(MAKEFILE_LIST))))

SOURCES = $(shell find $(MAKEFILE_DIR) -name "*.go" -not -path "./vendor/*")

#***********************************************************
all: build

build: $(SOURCES)
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o lipo-app_arm64 *.go
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o lipo-app_amd64 *.go
	@lipo -create -output lipo-app lipo-app_amd64 lipo-app_arm64

install: build
	install -m 644 lipo-app $(PREFIX) 

clean:
	@rm -rf lipo-app_arm64
	@rm -rf lipo-app_amd64
	@rm -rf lipo-app

