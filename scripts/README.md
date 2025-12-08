# Scripts để quản lý TikTok Access Token

Các script PowerShell này giúp bạn quản lý TikTok access tokens cho ứng dụng.

## Scripts có sẵn

### 1. `list_accounts.ps1`
Liệt kê tất cả accounts đã cấu hình.

**Usage:**
```powershell
.\list_accounts.ps1
```

**Output:** Danh sách tất cả accounts với:
- Account ID
- YouTube Channel ID
- TikTok Account ID
- Trạng thái Active/Inactive

### 2. `update_token.ps1`
Cập nhật token cho một account bằng Account ID.

**Usage:**
```powershell
# Với token được cung cấp
.\update_token.ps1 -AccountId <account_id> -NewToken <new_token>

# Hoặc prompt để nhập token (an toàn hơn)
.\update_token.ps1 -AccountId <account_id>
```

**Ví dụ:**
```powershell
.\update_token.ps1 -AccountId c139a639-3143-4058-960b-4fde6d1d9cae
```

### 3. `update_token_auto.ps1` ⭐
Tự động tìm account bằng TikTok Account ID và cập nhật token.

**Usage:**
```powershell
# Với token được cung cấp
.\update_token_auto.ps1 -TikTokAccountId <tiktok_account_id> -NewToken <new_token>

# Hoặc prompt để nhập token
.\update_token_auto.ps1 -TikTokAccountId <tiktok_account_id>
```

**Ví dụ:**
```powershell
.\update_token_auto.ps1 -TikTokAccountId 7580560736729088017
```

### 4. `fix_expired_token.ps1` ⭐⭐⭐ (Khuyến nghị cho token hết hạn)
Hướng dẫn từng bước để lấy authorization code và exchange để lấy token mới với refresh token.

**Usage:**
```powershell
.\fix_expired_token.ps1 -AccountId <account_id>
```

**Ví dụ:**
```powershell
.\fix_expired_token.ps1 -AccountId 6b749ea3-17ee-423b-bdd6-04a7ab62853d
```

**Lợi ích:**
- Tự động lấy token mới với refresh token
- Token sẽ tự động refresh khi hết hạn
- Hướng dẫn từng bước rõ ràng

## Cách sử dụng nhanh

1. **Kiểm tra accounts hiện có:**
   ```powershell
   .\list_accounts.ps1
   ```

2. **Cập nhật token (nếu biết TikTok Account ID):**
   ```powershell
   .\update_token_auto.ps1 -TikTokAccountId 7580560736729088017
   ```
   Nhập token mới khi được prompt.

3. **Hoặc cập nhật bằng Account ID:**
   ```powershell
   .\update_token.ps1 -AccountId c139a639-3143-4058-960b-4fde6d1d9cae
   ```

## Lưu ý

- Mặc định API server chạy tại `http://localhost:8080`
- Nếu server chạy ở port khác, thêm parameter `-ApiBaseUrl`
- Token sẽ được nhập dưới dạng secure string (ẩn khi gõ) nếu không cung cấp qua parameter

## Troubleshooting

### Lỗi: "Cannot connect to server"
- Đảm bảo ứng dụng đang chạy
- Kiểm tra port trong `config.yaml` (mặc định: 8080)
- Thử với `-ApiBaseUrl http://localhost:8080`

### Lỗi: "Account not found"
- Kiểm tra Account ID hoặc TikTok Account ID có đúng không
- Chạy `list_accounts.ps1` để xem danh sách accounts

### Lỗi: "Access token is invalid"
- Token có thể đã hết hạn ngay sau khi lấy
- Đảm bảo token có scope `video.upload`
- Lấy token mới từ TikTok Developer Portal

