BUILD_DIR = build
ENTRY = main.go
TARGET = $(BUILD_DIR)/sentry-node

all: run

$(TARGET): $(ENTRY)
	@go build -o $(TARGET) $(ENTRY)

run: $(TARGET)
	@./$(TARGET)

clean: 
	rm -rf $(BUILD_DIR)

.PHONY: all run clean
