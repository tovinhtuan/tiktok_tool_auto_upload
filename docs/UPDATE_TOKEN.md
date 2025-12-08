# Hướng dẫn cập nhật TikTok Access Token

Khi gặp lỗi "Access token is invalid or expired", bạn cần cập nhật token mới cho account.

## Cách 1: Sử dụng PowerShell Script (Khuyến nghị)

### Phương án A: Tự động tìm account bằng TikTok Account ID (Dễ nhất)

Nếu bạn biết TikTok Account ID (ví dụ: `7580560736729088017`):

```powershell
cd scripts
.\update_token_auto.ps1 -TikTokAccountId 7580560736729088017
```

Script sẽ tự động tìm account và prompt bạn nhập token mới.

### Phương án B: Sử dụng Account ID

#### Bước 1: Liệt kê tất cả accounts để lấy Account ID

```powershell
cd scripts
.\list_accounts.ps1
```

Kết quả sẽ hiển thị danh sách accounts với Account ID của mỗi account.

#### Bước 2: Lấy token mới từ TikTok

1. Truy cập URL sau để lấy authorization code:
   ```
   https://www.tiktok.com/v2/auth/authorize/?client_key=sbawggp68sw26gl2hv&scope=user.info.basic,video.upload&response_type=code&redirect_uri=https%3A%2F%2Ftovinhtuan.github.io%2Ftiktok-policy%2Fcallback&state=12345
   ```

2. Sau khi authorize, bạn sẽ được redirect về callback URL với `code` parameter
3. Sử dụng `code` để lấy access token từ TikTok API

#### Bước 3: Cập nhật token

```powershell
.\update_token.ps1 -AccountId <account_id> -NewToken <new_token>
```

Hoặc để script tự động prompt cho token:
```powershell
.\update_token.ps1 -AccountId <account_id>
```

Ví dụ:
```powershell
.\update_token.ps1 -AccountId c139a639-3143-4058-960b-4fde6d1d9cae
```

## Cách 2: Sử dụng HTTP API trực tiếp

### Bước 1: Lấy danh sách accounts

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/accounts" -Method GET
```

### Bước 2: Cập nhật token

```powershell
$body = @{
    tiktok_access_token = "your_new_token_here"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/accounts/c139a639-3143-4058-960b-4fde6d1d9cae" `
    -Method PATCH `
    -Headers @{"Content-Type"="application/json"} `
    -Body $body
```

## Cách 3: Sử dụng cURL (nếu có)

```bash
# List accounts
curl http://localhost:8080/api/accounts

# Update token
curl -X PATCH http://localhost:8080/api/accounts/c139a639-3143-4058-960b-4fde6d1d9cae \
  -H "Content-Type: application/json" \
  -d '{"tiktok_access_token":"your_new_token_here"}'
```

## Cách 4: Cập nhật qua config.yaml và restart

Nếu account được bootstrap từ `config.yaml`:

1. Mở file `cmd/config.yaml`
2. Tìm account tương ứng và cập nhật `tiktok_access_token`
3. Restart ứng dụng

Lưu ý: Cách này chỉ hoạt động nếu account được tạo từ config.yaml khi bootstrap.

## Kiểm tra token mới

Sau khi cập nhật, ứng dụng sẽ tự động validate token khi upload video tiếp theo. 
Bạn có thể kiểm tra log để xác nhận token đã được cập nhật thành công.

## Lưu ý quan trọng

- TikTok access tokens thường có thời hạn ngắn (vài giờ đến vài ngày)
- Token cần có scope `video.upload` để upload video
- Nếu token hết hạn thường xuyên, hãy xem xét implement token refresh mechanism

