# Build stage
FROM python:3.11-slim AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# Runtime stage
FROM python:3.11-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libpq5 \
    && rm -rf /var/lib/apt/lists/*

# Copy Python packages from builder
COPY --from=builder /root/.local /root/.local

# Copy application
COPY serving ./serving

# Create models directory
RUN mkdir -p /models

# Create non-root user
RUN useradd -r -s /bin/false appuser && chown -R appuser:appuser /app /models
USER appuser

# Add local packages to PATH
ENV PATH=/root/.local/bin:$PATH

EXPOSE 8000

CMD ["python", "-m", "serving.main"]