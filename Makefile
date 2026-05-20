BIN := agno-docs-pp-cli
TAGS := sqlite_fts5
VERSION ?= v0.1.0
LDFLAGS := -X agno-docs-pp-cli/internal/cli.Version=$(VERSION)

.PHONY: build install sync doctor clean test

build:
	go build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/$(BIN)

install:
	go install -tags "$(TAGS)" -ldflags "$(LDFLAGS)" ./cmd/$(BIN)

sync: install
	$(BIN) sync

doctor:
	$(BIN) doctor

test:
	go test -tags "$(TAGS)" ./...

clean:
	rm -f $(BIN)
	rm -f $(HOME)/.local/share/$(BIN)/data.db
