BINARY_NAME=opencode_skill
INSTALL_DIR=$(HOME)/bin

.PHONY: all build install restart clean

all: build

build:
	go build -o $(BINARY_NAME)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"
	@echo "Restarting daemon..."
	@$(INSTALL_DIR)/$(BINARY_NAME) restart &

restart:
	@echo "Restarting daemon..."
	@$(INSTALL_DIR)/$(BINARY_NAME) restart &

clean:
	rm -f $(BINARY_NAME)
	rm -f /tmp/opencode_wrapper_go.log
