GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTIDY=$(GOCMD) mod tidy
GORUN=$(GOCMD) run

BINARY_NAME=go-cracker
BINARY_UNIX=$(BINARY_NAME)

all: run

.PHONY: all run build clean tidy

run:
	@echo "Running the application..."
	$(GORUN) .

build:
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

