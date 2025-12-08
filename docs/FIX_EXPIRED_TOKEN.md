# Hướng dẫn khắc phục lỗi Token hết hạn

Khi gặp lỗi "Access token is invalid or expired and no refresh token available", bạn cần cập nhật token mới bằng cách exchange authorization code.

## Cách nhanh nhất: Sử dụng Script

```powershell
cd scripts
.\fix_expired_token.ps1 -AccountId 6b749ea3-17ee-423b-bdd6-04a7ab62853d
```

Script sẽ hướng dẫn bạn từng bước.

## Cách thủ công

### Bước 1: Lấy Authorization Code

1. Mở URL sau trong browser:
   ```
   https://www.tiktok.com/v2/auth/authorize/?client_key=sbawggp68sw26gl2hv&scope=user.info.basic,video.upload&response_type=code&redirect_uri=https%3A%2F%2Ftovinhtuan.github.io%2Ftiktok-policy%2Fcallback&state=12345
   ```

2. Đăng nhập và authorize ứng dụng

3. Sau khi authorize, bạn sẽ được redirect về URL dạng:
   ```
   https://tovinhtuan.github.io/tiktok-policy/callback?code=YOUR_CODE_HERE&scopes=user.info.basic,video.upload&state=12345
   ```

4. Copy toàn bộ URL callback (script sẽ tự trích `code` từ query string)

### Bước 2: Exchange Code để lấy Token

#### Sử dụng PowerShell:

```powershell
$body = @{
    code = "YOUR_CODE_HERE"
    redirect_uri = "https://tovinhtuan.github.io/tiktok-policy/callback"
    account_id = "6b749ea3-17ee-423b-bdd6-04a7ab62853d"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/tiktok/exchange-code" `
    -Method POST `
    -Headers @{"Content-Type"="application/json"} `
    -Body $body
```

#### Sử dụng cURL:

```bash
curl -X POST http://localhost:8080/api/tiktok/exchange-code \
  -H "Content-Type: application/json" \
  -d '{
    "code": "YOUR_CODE_HERE",
    "redirect_uri": "https://tovinhtuan.github.io/tiktok-policy/callback",
    "account_id": "6b749ea3-17ee-423b-bdd6-04a7ab62853d"
  }'
```

#### Sử dụng Postman/Insomnia:

- Method: `POST`
- URL: `http://localhost:8080/api/tiktok/exchange-code`
- Headers: `Content-Type: application/json`
- Body (JSON):
  ```json
  {
    "code": "YOUR_CODE_HERE",
    "redirect_uri": "https://tovinhtuan.github.io/tiktok-policy/callback",
    "account_id": "6b749ea3-17ee-423b-bdd6-04a7ab62853d"
  }
  ```

### Bước 3: Kiểm tra kết quả

Response sẽ có dạng:
```json
{
  "status": "success",
  "account": { ... },
  "expires_in": 7200,
  "has_refresh_token": true,
  "token_type": "Bearer",
  "scope": "user.info.basic,video.upload"
}
```

Nếu `has_refresh_token` là `true`, token sẽ tự động refresh khi hết hạn.

## Tìm Account ID

Nếu bạn không biết Account ID, có thể:

1. Sử dụng script list accounts:
   ```powershell
   .\scripts\list_accounts.ps1
   ```

2. Hoặc gọi API:
   ```powershell
   Invoke-RestMethod -Uri "http://localhost:8080/api/accounts" -Method GET
   ```

3. Tìm account theo TikTok Account ID:
   ```powershell
   .\scripts\update_token_auto.ps1 -TikTokAccountId YOUR_TIKTOK_ACCOUNT_ID
   ```

## Lưu ý quan trọng

- **Authorization code chỉ dùng được 1 lần** - sau khi exchange, code sẽ không còn hiệu lực
- **Code có thời hạn ngắn** - thường chỉ vài phút, nên exchange ngay sau khi lấy được
- **Refresh token** - nếu TikTok API trả về refresh token, hệ thống sẽ tự động refresh khi token hết hạn
- **Nếu không có refresh token** - bạn sẽ cần lặp lại quá trình này khi token hết hạn

## Troubleshooting

### Lỗi: "code is invalid"
- Code đã được sử dụng hoặc hết hạn
- Lấy code mới từ TikTok

### Lỗi: "account not found"
- Kiểm tra Account ID có đúng không
- Sử dụng `list_accounts.ps1` để xem danh sách accounts

### Lỗi: "failed to exchange code"
- Kiểm tra code có đúng không
- Kiểm tra redirect_uri có khớp với URL trong authorization request không
- Kiểm tra TikTok API có đang hoạt động không

