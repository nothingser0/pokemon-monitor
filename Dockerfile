# ============================================================
# Pokemon GO Coordinate Monitor - Docker Image
# ============================================================
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install dependencies first (better layer caching)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY monitor.py .
COPY config.json .

# Create a non-root user and a writable data directory (for state file)
RUN useradd --create-home --uid 1000 appuser \
    && mkdir -p /app/data \
    && chown -R appuser:appuser /app/data

# Run as non-root user
USER appuser

# Run the monitor in continuous loop mode
CMD ["python", "monitor.py"]
