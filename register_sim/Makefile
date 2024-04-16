APP_NAME := register_sim

VERSION ?= latest

BINARIES := \
	bin/$(APP_NAME)-$(VERSION)-linux-amd64 \
	bin/$(APP_NAME)-$(VERSION)-linux-arm64 \
	bin/$(APP_NAME)-$(VERSION)-darwin-amd64 \
	bin/$(APP_NAME)-$(VERSION)-darwin-arm64 \
	bin/$(APP_NAME)-$(VERSION)-windows-amd64.exe \
	bin/$(APP_NAME)-$(VERSION)-windows-arm64.exe

all: $(BINARIES)

bin/$(APP_NAME)-$(VERSION)-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o $@

bin/$(APP_NAME)-$(VERSION)-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o $@

bin/$(APP_NAME)-$(VERSION)-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $@

bin/$(APP_NAME)-$(VERSION)-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $@

bin/$(APP_NAME)-$(VERSION)-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 go build -o $@

bin/$(APP_NAME)-$(VERSION)-windows-arm64.exe:
	GOOS=windows GOARCH=arm64 go build -o $@

zip: all
	zip -j bin/$(APP_NAME)-$(VERSION)-all.zip $(BINARIES)

clean:
	rm -r bin

.PHONY: all clean
