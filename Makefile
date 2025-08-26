GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTIDY=$(GOCMD) mod tidy
GORUN=$(GOCMD) run
GOINIT=$(GOCMD) mod init

BINARY_NAME=go-cracker
MODULE_NAME=go-cracker

all: run

.PHONY: all run build clean tidy init

init:
	@if [ ! -f go.mod ]; then \
		echo "Initializing Go module..."; \
		$(GOINIT) $(MODULE_NAME); \
		$(GOTIDY); \
	fi

run: build
	@echo "Running the application..."
	./$(BINARY_NAME)

build: init
	@echo "Building the application..."
	$(GOBUILD) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	@echo "Cleanup complete."

tidy:
	@echo "Tidying Go module dependencies..."
	$(GOTIDY)
	@echo "Dependencies tidied."
