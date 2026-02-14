BINARY_NAME=opencode_wrapper
INSTALL_DIR=$(HOME)/bin

.PHONY: all build install clean

all: build

build:
	go build -o $(BINARY_NAME)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"

clean:
	rm -f $(BINARY_NAME)
	rm -f /tmp/opencode_wrapper_go.log
