BIN          := gh-kanban
PKG          := ./cmd/kanban
EXT_NAME     := gh-kanban
EXT_DIR      := $(HOME)/.local/share/gh/extensions/$(EXT_NAME)

.PHONY: build install uninstall reinstall run test vet tidy clean

build:
	go build -o $(BIN) $(PKG)

install: build
	mkdir -p $(EXT_DIR)
	cp $(BIN) $(EXT_DIR)/$(EXT_NAME)
	@echo "installed to $(EXT_DIR)/$(EXT_NAME)"
	@echo "run: gh kanban"

uninstall:
	rm -rf $(EXT_DIR)
	@echo "removed $(EXT_DIR)"

reinstall: uninstall install

run:
	go run $(PKG)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN)
