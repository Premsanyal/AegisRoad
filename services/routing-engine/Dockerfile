# Build stage
FROM rust:1.82-slim as builder

WORKDIR /app

# Install nightly Rust for edition2024 support
RUN rustup toolchain install nightly --component rustfmt clippy
RUN rustup default nightly

# Install system dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Copy manifests
COPY Cargo.toml ./

# Copy proto files for protobuf generation
COPY proto ./proto

# Copy build.rs for protobuf generation
COPY build.rs ./

# Create dummy source to cache dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo generate-lockfile && cargo build --release
RUN rm -rf src

# Copy actual source
COPY src ./src

# Build with protobuf generation
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

# Copy binary
COPY --from=builder /app/target/release/routing-engine /app/routing-engine

# Copy profiles
COPY profiles /profiles

# Create non-root user
RUN useradd -r -s /bin/false appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 50051

CMD ["/app/routing-engine"]