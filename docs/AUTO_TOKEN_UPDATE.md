# Hướng dẫn cập nhật Token tự động

Hệ thống đã được tự động hóa hoàn toàn! Bạn chỉ cần click và authorize, không cần copy/paste code hay gọi API.

## Cách sử dụng (Rất đơn giản!)

### Bước 1: Mở Web UI
Mở browser và truy cập:
```
http://localhost:8080/
```

### Bước 2: Click "Authorize & Update Token"
- Bạn sẽ thấy danh sách tất cả accounts
- Tìm account cần update token
- Click nút **"🔑 Authorize & Update Token"**

### Bước 3: Authorize trên TikTok
- Browser sẽ tự động redirect đến TikTok
- Đăng nhập và authorize ứng dụng
- Sau khi authorize, bạn sẽ được redirect về lại

### Bước 4: Xong!
- Hệ thống tự động:
  - Nhận code từ TikTok
  - Exchange code để lấy access token + refresh token
  - Cập nhật account với token mới
  - Hiển thị kết quả thành công

## Lưu ý quan trọng

### Redirect URI Configuration

Hệ thống sử dụng redirect URI: `http://localhost:8080/api/tiktok/callback`

**Bạn cần đảm bảo TikTok app có redirect URI này:**

1. Vào [TikTok Developer Portal](https://developers.tiktok.com/)
2. Chọn app của bạn
3. Vào phần "Redirect URI" hoặc "OAuth Settings"
4. Thêm redirect URI: `http://localhost:8080/api/tiktok/callback`
5. Save changes

**Nếu không thêm redirect URI này, TikTok sẽ từ chối callback và hiển thị lỗi "redirect_uri_mismatch".**

### Alternative: Sử dụng redirect URI hiện có

Nếu bạn đã có redirect URI khác (ví dụ: `https://tovinhtuan.github.io/tiktok-policy/callback`), bạn có thể:

1. Sử dụng endpoint `/api/tiktok/exchange-code` với redirect URI đó
2. Hoặc cập nhật code để sử dụng redirect URI của bạn

## Troubleshooting

### Lỗi: "redirect_uri_mismatch"
- **Nguyên nhân:** Redirect URI trong request không khớp với redirect URI đã đăng ký trong TikTok app
- **Giải pháp:** Thêm `http://localhost:8080/api/tiktok/callback` vào TikTok app settings

### Lỗi: "Account not found"
- Kiểm tra Account ID có đúng không
- Sử dụng Web UI để xem danh sách accounts

### Lỗi: "Failed to exchange code"
- Code có thể đã hết hạn (thường chỉ vài phút)
- Thử lại từ đầu (click "Authorize" lại)

### Web UI không mở được
- Đảm bảo ứng dụng đang chạy
- Kiểm tra port trong `config.yaml` (mặc định: 8080)
- Thử: `http://localhost:8080/api/health` để kiểm tra server

## So sánh với cách cũ

**Cách cũ (thủ công):**
1. Mở URL authorization
2. Copy code từ callback URL
3. Gọi API exchange-code với code
4. Kiểm tra kết quả

**Cách mới (tự động):**
1. Click "Authorize" trên Web UI
2. Authorize trên TikTok
3. Xong! ✅

## API Endpoints

Nếu bạn muốn tự động hóa hơn nữa, có thể sử dụng API trực tiếp:

- `GET /api/tiktok/authorize/{account_id}` - Bắt đầu OAuth flow
- `GET /api/tiktok/callback` - Callback endpoint (tự động xử lý)
- `POST /api/tiktok/exchange-code` - Exchange code manually (nếu cần)


