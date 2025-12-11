#!/bin/sh
set -e

# Change to app directory
cd /app || exit 1

# Create config.yaml from environment variables if it doesn't exist
if [ ! -f config.yaml ]; then
  echo "Creating config.yaml from environment variables..."

  # Get PORT from environment (Render sets this)
  PORT=${PORT:-8080}
  
  # Create config.yaml with environment variable support
  cat > config.yaml <<EOF
server:
  port: "${PORT}"

youtube:
  api_key: "${YOUTUBE_API_KEY}"

tiktok:
  api_key: "${TIKTOK_API_KEY}"
  api_secret: "${TIKTOK_API_SECRET}"
  region: "${TIKTOK_REGION:-JP}"
  base_url: "${TIKTOK_BASE_URL:-https://open-api.tiktok.com}"
  upload_init_path: "${TIKTOK_UPLOAD_INIT_PATH:-/video/upload/}"
  publish_path: "${TIKTOK_PUBLISH_PATH:-/video/publish/}"
  redirect_uri: "${TIKTOK_REDIRECT_URI:-}"

cron:
  schedule: "${CRON_SCHEDULE:-*/5 * * * *}"

download:
  dir: "${DOWNLOAD_DIR:-./downloads}"
  max_concurrent: ${DOWNLOAD_MAX_CONCURRENT:-5}
  timeout: "${DOWNLOAD_TIMEOUT:-10m}"
  buffer_size: ${DOWNLOAD_BUFFER_SIZE:-1048576}
  yt_dlp_path: "${YT_DLP_PATH:-}"

upload:
  max_concurrent: ${UPLOAD_MAX_CONCURRENT:-3}
  timeout: "${UPLOAD_TIMEOUT:-15m}"
  buffer_size: ${UPLOAD_BUFFER_SIZE:-1048576}

database:
  url: "${DATABASE_URL:-sqlite3:./data.db}"

performance:
  worker_pool_size: ${WORKER_POOL_SIZE:-0}
  http_client_timeout: "${HTTP_CLIENT_TIMEOUT:-30s}"
  max_idle_conns: ${MAX_IDLE_CONNS:-200}
  max_conns_per_host: ${MAX_CONNS_PER_HOST:-50}
  max_concurrent_io: ${MAX_CONCURRENT_IO:-8}

logging:
  dir: "${LOG_DIRECTORY:-./logs}"
  output_file: "${LOG_OUTPUT_FILE:-app.log}"
  error_file: "${LOG_ERROR_FILE:-app.error.log}"
EOF

  # Add bootstrap accounts if environment variables are set
  if [ -n "$BOOTSTRAP_ACCOUNTS" ]; then
    echo "" >> config.yaml
    echo "accounts:" >> config.yaml
    echo "$BOOTSTRAP_ACCOUNTS" | tr ';' '\n' | while IFS=',' read -r youtube_id tiktok_id token active; do
      if [ -n "$youtube_id" ] && [ -n "$tiktok_id" ]; then
        echo "  - youtube_channel_id: \"${youtube_id}\"" >> config.yaml
        echo "    tiktok_account_id: \"${tiktok_id}\"" >> config.yaml
        if [ -n "$token" ]; then
          echo "    tiktok_access_token: \"${token}\"" >> config.yaml
        fi
        if [ -n "$active" ]; then
          echo "    is_active: ${active}" >> config.yaml
        else
          echo "    is_active: true" >> config.yaml
        fi
      fi
    done
  fi

  echo "Config file created at config.yaml"
else
  echo "Using existing config.yaml"
  # Still update PORT if set via environment
  if [ -n "$PORT" ]; then
    echo "Updating server port to ${PORT} in config.yaml"
    # Simple sed replacement for port in config.yaml
    sed -i "s/^  port:.*/  port: \"${PORT}\"/" config.yaml 2>/dev/null || true
  fi
fi

# Ensure required directories exist
mkdir -p downloads logs

# Decode YouTube cookies from base64 if provided
if [ -n "$YOUTUBE_COOKIES_BASE64" ]; then
  echo "Decoding YouTube cookies from environment variable..."
  echo "$YOUTUBE_COOKIES_BASE64" | base64 -d > youtube_cookies.txt
  if [ $? -eq 0 ]; then
    echo "YouTube cookies decoded successfully"
  else
    echo "Warning: Failed to decode YouTube cookies"
  fi
fi

# Execute the main application
exec ./auto_upload_tiktok

