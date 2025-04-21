APP_NAME := myrient-cli
SRC_DIR := src
OUTPUT_DIR := builds

PLATFORMS := \
  linux/amd64 \
  linux/arm64 \
  linux/arm \
  darwin/amd64 \
  darwin/arm64 \
  windows/amd64 \
  windows/arm64

all: clean build

build:
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		OS=$${platform%%/*}; \
		ARCH=$${platform##*/}; \
		EXT=$$( [ "$$OS" = "windows" ] && echo ".exe" || echo "" ); \
		OUTPUT=$(OUTPUT_DIR)/$(APP_NAME)-$${OS}-$${ARCH}$${EXT}; \
		echo "➡️  Building $$OS/$$ARCH → $$OUTPUT"; \
		GOOS=$$OS GOARCH=$$ARCH go build -C $(SRC_DIR) -o ../$$OUTPUT .; \
	done
	@echo "✅ Build complete."

clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(OUTPUT_DIR)
