FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# Copy go mod and sum files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY *.go ./

# Build the Go app statically
# Output the binary as 'steamdeck-checker' within /app
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o steamdeck-checker .


# --- Final Stage ---
FROM debian:bullseye-slim

# Install necessary dependencies for Chromium and fonts
RUN rm -rf /var/lib/apt/lists/* \
    && apt-get update \
    && apt-get install -y --no-install-recommends --fix-missing \
    ca-certificates \
    chromium \
    && rm -rf /var/lib/apt/lists/*


# Set the working directory
WORKDIR /app

# Copy the built Go binary from the builder stage
COPY --from=builder /app/steamdeck-checker .

# Make the binary executable (good practice)
RUN chmod +x ./steamdeck-checker

# Command to run the executable
# The app will run Chromium in the background
CMD ["./steamdeck-checker"]
