BINARY_NAME=opencode_skill
INSTALL_DIR=$(HOME)/bin

.PHONY: all build install stop start restart clean

all: build

build:
	go build -o $(BINARY_NAME)

stop:
	@echo "Stopping daemon..."
	@$(INSTALL_DIR)/$(BINARY_NAME) stop 2>/dev/null || true

start:
	@echo "Starting daemon..."
	@$(INSTALL_DIR)/$(BINARY_NAME) start

install: build stop
	@echo "Removing old binary..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installing..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"
	@$(MAKE) start

restart:
	@echo "Restarting daemon..."
	@$(INSTALL_DIR)/$(BINARY_NAME) restart

clean:
	rm -f $(BINARY_NAME)
	rm -f /tmp/opencode_skill_go.log
