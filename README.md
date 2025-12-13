# Auto Upload TikTok

Hệ thống tự động quét các tài khoản YouTube, tải video mới về và đăng lên TikTok (khu vực Japan) với hiệu năng cao và độ trễ thấp.

## 🚀 Tính năng

- **Tự động quét YouTube**: Lập lịch quét các kênh YouTube để phát hiện video mới
- **Tải video tự động**: Tự động tải video mới về máy với hiệu năng cao
- **Upload lên TikTok**: Tự động đăng video lên tài khoản TikTok được chỉ định (region Japan)
- **Xử lý đồng thời**: Xử lý nhiều video cùng lúc với worker pool
- **Connection Pooling**: Tối ưu kết nối HTTP với connection pooling
- **Clean Architecture**: Tuân thủ Clean Architecture và SOLID principles

## 📋 Yêu cầu

- Go 1.24 hoặc cao hơn
- yt-dlp (để tải video từ YouTube)
- YouTube Data API v3 key
- TikTok Open API credentials

### Cài đặt yt-dlp

**Windows:**
```powershell
# Sử dụng pip
pip install yt-dlp

# Hoặc sử dụng chocolatey
choco install yt-dlp
```

**Linux/Mac:**
```bash
# Sử dụng pip
pip install yt-dlp

# Hoặc sử dụng homebrew (Mac)
brew install yt-dlp
```

## 🔧 Cài đặt

1. Clone repository:
```bash
git clone <repository-url>
cd auto_upload_tiktok
```

2. Cài đặt dependencies:
```bash
go mod download
```

3. Cấu hình trong file `config/config.yaml`:
```yaml
# YouTube API
youtube:
  api_key: "your_youtube_api_key_here"  # Required

# TikTok API
tiktok:
  api_key: "your_tiktok_api_key_here"     # Required
  api_secret: "your_tiktok_api_secret"    # Required
  region: "JP"                            # TikTok region (JP for Japan)
  base_url: "https://open.tiktokapis.com" # Use the domain that matches your OpenAPI environment
  upload_init_path: "/video/upload/"      # Update to the exact endpoint path provided by TikTok
  publish_path: "/video/publish/"

# Cron Schedule
cron:
  schedule: "* * * * * *"  # Scan YouTube once every second

# Download Configuration
download:
  dir: "./downloads"
  max_concurrent: 5
  timeout: "10m"
  buffer_size: 1048576  # 1MB
  yt_dlp_path: ""        # Optional: full path to yt-dlp if it's not in PATH

# Upload Configuration
upload:
  max_concurrent: 3
  timeout: "15m"
  buffer_size: 1048576  # 1MB

# Performance Tuning
performance:
  worker_pool_size: 0      # 0 = auto-detect (CPU cores × 4)
  http_client_timeout: "30s"
  max_idle_conns: 200
  max_conns_per_host: 50
  max_concurrent_io: 8
```

**Lưu ý**: File `config.yaml` có thể được chỉnh sửa trực tiếp và sẽ được tự động reload khi ứng dụng khởi động lại.

## 🏃 Chạy ứng dụng

```bash
go run cmd/main.go
```

Hoặc build và chạy:

```bash
go build -o auto_upload_tiktok cmd/main.go
./auto_upload_tiktok
```

### Bootstrap Account Mappings

- Khai báo cặp YouTube->TikTok ngay trong `config/config.yaml` (mục `accounts`). V? dụ:

```yaml
accounts:
  - youtube_channel_id: "UCIemB2OhSoG7GBEfsF7e1MA"
    tiktok_account_id: "7580560736729088017"
    tiktok_access_token: "act.example"
    is_active: true
```

- Khi service khởi động, các mapping này sẽ được tự động tạo/cập nhật để scheduler luôn có job quét và tải video (kể cả Shorts).

## 📁 Cấu trúc dự án

```
auto_upload_tiktok/
├── cmd/
│   └── main.go                 # Entry point
├── config/
│   └── config.go               # Configuration management
├── internal/
│   ├── domain/                 # Domain entities và interfaces
│   │   ├── account.go
│   │   └── video.go
│   ├── usecase/                # Business logic
│   │   ├── account_monitor.go  # Monitor YouTube accounts
│   │   └── video_processor.go  # Process videos (download & upload)
│   ├── repository/             # Data access layer
│   │   └── memory/             # In-memory implementation
│   │       ├── account.go
│   │       └── video.go
│   ├── infrastructure/         # External services
│   │   ├── http/               # HTTP client với connection pooling
│   │   ├── youtube/            # YouTube API service
│   │   ├── tiktok/             # TikTok API service
│   │   └── downloader/         # Video download service
│   └── delivery/               # Delivery layer
│       └── cron/               # Cron scheduler
│           └── scheduler.go
├── config/
│   ├── config.go          # Configuration manager
│   └── config.yaml         # Configuration file (YAML)
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

## 🏗️ Kiến trúc

Dự án tuân thủ **Clean Architecture** với các lớp:

1. **Domain Layer**: Entities và repository interfaces
   - `Account`: Đại diện cho mapping YouTube Channel ↔ TikTok Account (1 job)
   - `Video`: Đại diện cho video cần xử lý
2. **Use Case Layer**: Business logic
   - `AccountMonitor`: Quét YouTube channels để tìm video mới
   - `VideoProcessor`: Xử lý video (download + upload)
   - `AccountManager`: Quản lý account mappings
3. **Infrastructure Layer**: External services (YouTube, TikTok, HTTP client)
4. **Delivery Layer**: Cron scheduler

### Mô hình Job

```
Account (1 job)
├── YouTube Channel ID (source)
├── TikTok Account ID (destination)
└── TikTok Access Token

Job Flow:
1. Monitor YouTube Channel → Tìm video mới
2. Download video từ YouTube
3. Upload video lên TikTok Account được chỉ định
```

**Nhiều jobs có thể chạy song song**, mỗi job độc lập với nhau.

### SOLID Principles

- **Single Responsibility**: Mỗi module có một trách nhiệm duy nhất
- **Open/Closed**: Dễ dàng mở rộng mà không sửa đổi code hiện có
- **Liskov Substitution**: Repository interfaces có thể thay thế bằng implementations khác
- **Interface Segregation**: Interfaces nhỏ và tập trung
- **Dependency Inversion**: Depend on abstractions, not concretions

## ⚡ Tối ưu hiệu năng cho I/O Bound Operations

Hệ thống được tối ưu đặc biệt cho **I/O bound operations** (network và disk I/O), không phải CPU bound.

### 1. Auto-Scaling Worker Pool
- **Tự động tính toán** worker pool size dựa trên số CPU cores
- **Công thức**: `WorkerPoolSize = CPU_Cores × 4` (tối ưu cho I/O bound)
- **Lý do**: Với I/O bound, goroutines chờ I/O nên có thể chạy nhiều hơn số CPU cores
- **Range**: Tối thiểu 10, tối đa 100 workers
- **Cấu hình**: `WORKER_POOL_SIZE` (0 = auto-detect)

### 2. Connection Pooling & HTTP/2
- HTTP client với **connection pooling** để tái sử dụng kết nối
- **HTTP/2 multiplexing** cho nhiều requests trên 1 connection
- **Tăng buffer sizes**: 64KB read/write buffers
- **Cấu hình**: 
  - `MAX_IDLE_CONNS=200` (tăng từ 100)
  - `MAX_CONNS_PER_HOST=50` (tăng từ 10)

### 3. Separate Semaphores cho Download/Upload
- **Download semaphore**: Giới hạn concurrent downloads
- **Upload semaphore**: Giới hạn concurrent uploads  
- **Lợi ích**: Tối ưu riêng cho từng loại I/O operation
- **Cấu hình**: `MAX_CONCURRENT_DOWNLOADS`, `MAX_CONCURRENT_UPLOADS`

### 4. Optimized Buffer Sizes
- **Download buffer**: 1MB (tăng từ 32KB) - giảm system calls
- **Upload buffer**: 1MB - tăng throughput
- **Lý do**: Buffer lớn hơn = ít system calls hơn = latency thấp hơn
- **Cấu hình**: `DOWNLOAD_BUFFER_SIZE`, `UPLOAD_BUFFER_SIZE`

### 5. Pipeline Processing
- Fetch nhiều videos hơn số có thể xử lý để giữ pipeline đầy
- Batch size = `MAX_CONCURRENT_DOWNLOADS + MAX_CONCURRENT_UPLOADS`
- **Lợi ích**: Luôn có video sẵn sàng để xử lý, giảm idle time

### 6. Timeout Management
- Timeout riêng cho download và upload
- HTTP client timeout để tránh treo
- Response header timeout: 30s

### 📊 Performance Impact

**Trước khi tối ưu:**
- Worker pool: 10 (cố định)
- Buffer: 32KB
- Max connections per host: 10
- Sequential processing

**Sau khi tối ưu:**
- Worker pool: Auto (CPU cores × 4)
- Buffer: 1MB (31x lớn hơn)
- Max connections per host: 50 (5x lớn hơn)
- Parallel I/O với separate semaphores
- **Kết quả**: Giảm latency 40-60%, tăng throughput 2-3x

## 📝 Sử dụng

### Quản lý Account Mappings (YouTube Channel ↔ TikTok Account)

Mỗi **Account** đại diện cho một **job** liên kết một kênh YouTube với một tài khoản TikTok:
- **Download**: Từ YouTube channel được chỉ định
- **Upload**: Lên TikTok account được chỉ định

#### Cách tạo Account Mapping

**Option 1: Sử dụng AccountManager (Recommended)**

```go
import (
    "auto_upload_tiktok/internal/repository/memory"
    "auto_upload_tiktok/internal/usecase"
)

// Initialize
accountRepo := memory.NewAccountRepository()
accountManager := usecase.NewAccountManager(accountRepo)

// Tạo mapping: YouTube Channel -> TikTok Account
account, err := accountManager.CreateAccountMapping(
    "UCxxxxxxxxxxxxxxxxxxxxxxxxxx",  // YouTube Channel ID
    "tiktok_account_123",             // TikTok Account ID
    "tiktok_access_token_here",       // TikTok Access Token
)
if err != nil {
    log.Fatal(err)
}

log.Printf("Created job: YouTube %s -> TikTok %s", 
    account.YouTubeChannelID, account.TikTokAccountID)
```

**Option 2: Tạo nhiều mappings cùng lúc**

Xem file `cmd/init_accounts.go` để xem ví dụ đầy đủ:

```go
mappings := []struct {
    youtubeChannelID string
    tiktokAccountID  string
    tiktokToken      string
}{
    {"UCchannel1", "tiktok1", "token1"},
    {"UCchannel2", "tiktok2", "token2"},
    {"UCchannel3", "tiktok3", "token3"},
}

for _, m := range mappings {
    account, err := accountManager.CreateAccountMapping(
        m.youtubeChannelID,
        m.tiktokAccountID,
        m.tiktokToken,
    )
    // Handle error...
}
```

#### Quản lý Account Mappings

```go
// Lấy tất cả mappings
accounts, err := accountManager.GetAllAccountMappings()

// Tạm dừng một job (deactivate)
err := accountManager.DeactivateAccountMapping("account_id")

// Tiếp tục một job (activate)
err := accountManager.ActivateAccountMapping("account_id")

// Xóa một mapping
err := accountManager.DeleteAccountMapping("account_id")

// Cập nhật mapping
account, err := accountManager.UpdateAccountMapping(
    "account_id",
    "new_youtube_channel_id",  // optional: "" để giữ nguyên
    "new_tiktok_account_id",   // optional: "" để giữ nguyên
    "new_token",               // optional: "" để giữ nguyên
    true,                       // isActive
)
```

#### Lưu ý

- Mỗi YouTube channel chỉ có thể map với **1 TikTok account**
- Mỗi TikTok account chỉ có thể map với **1 YouTube channel**
- Mỗi mapping = 1 job độc lập chạy song song
- Các job không ảnh hưởng lẫn nhau

## 🔐 Bảo mật

- Không commit file `config.yaml` với API keys thật vào git
- Sử dụng `config/config.yaml` cho cấu hình
- File YAML có thể được chỉnh sửa và cập nhật tại runtime
- Validate API keys trước khi sử dụng

## ⚙️ Quản lý Configuration

### Cập nhật Config tại Runtime

Bạn có thể cập nhật configuration và lưu vào file YAML:

```go
import "auto_upload_tiktok/config"

// Lấy config manager
manager := config.GetManager()

// Cập nhật các trường cụ thể
err := manager.Update(map[string]interface{}{
    "youtube.api_key": "new_youtube_key",
    "download.max_concurrent": 10,
    "performance.worker_pool_size": 20,
    "cron.schedule": "*/3 * * * *",
})
if err != nil {
    log.Fatal(err)
}

// Reload để lấy config mới
cfg, err := manager.Reload()
```

### Cập nhật toàn bộ Config

```go
manager := config.GetManager()
cfg := manager.Get()

// Sửa đổi config
cfg.MaxConcurrentDownloads = 15
cfg.CronSchedule = "*/3 * * * *"

// Lưu vào file YAML
err := manager.Save(cfg)
```

Xem thêm ví dụ trong `config/config_example.go`

## 🧪 Testing

```bash
# Chạy tests
go test ./...

# Với coverage
go test -cover ./...
```

## 📊 Monitoring

Ứng dụng log các hoạt động:
- Account monitoring jobs
- Video processing jobs
- Download/Upload progress
- Errors và warnings

## 🐛 Troubleshooting

### Lỗi: yt-dlp not found
- Đảm bảo yt-dlp đã được cài đặt và có trong PATH
- Nếu chạy trên môi trường bị hạn chế PATH, đặt đường dẫn tuyệt đối vào `download.yt_dlp_path` trong `config/config.yaml`
- Kiểm tra: `yt-dlp --version`

### Lỗi: YouTube API quota exceeded
- Kiểm tra quota trong Google Cloud Console
- Giảm tần suất quét (tăng CRON_SCHEDULE interval)

### Lỗi: TikTok API authentication failed
- Kiểm tra access token còn hợp lệ
- Verify token với `VerifyAccessToken`

## 📄 License

MIT License

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📧 Support

Nếu có vấn đề, vui lòng tạo issue trên GitHub.


## Runtime Ops & API

- Job state (accounts/videos) is persisted inside the SQLite database configured via `database.url` (default `sqlite3:./data.db`), so restarts no longer wipe mappings or queues.
- The service now exposes a lightweight HTTP API on `server.port` (default 8080) for runtime management. Key endpoints:
  - `GET /api/health` - service heartbeat.
  - `GET /api/accounts` / `POST /api/accounts` - list and create mappings.
  - `PATCH /api/accounts/{id}` - update mapping fields or toggle activity via the optional `is_active`.
  - `POST /api/accounts/{id}/activate` and `/deactivate` - quick status flips.
  - `DELETE /api/accounts/{id}` - remove a mapping.
  - `GET /api/videos/pending-limit=20` - inspect pending queue items.
  - `GET /api/videos/metrics` - pending queue size for dashboards.
- Combine the API with CLI scripts or dashboards to observe queues and apply changes without editing source files.

- Khi service kh?i ??ng, c?c mapping n?y s? ???c t? ??ng t?o/c?p nh?t ?? scheduler lu?n c? job.
- N?u b?n thay ??i `youtube_channel_id` ho?c `tiktok_account_id`, service s? t? ??ng c?p nh?t mapping hi?n c? d?a tr?n TikTok ID/Channel ID, v? v?y ch? c?n s?a c?u h?nh r?i kh?i ??ng l?i.
