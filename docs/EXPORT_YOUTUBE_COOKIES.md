# Hướng dẫn Export YouTube Cookies để bypass bot detection

## Tại sao cần cookies?

YouTube tin tưởng requests từ browsers đã đăng nhập hơn là anonymous requests. Bằng cách export cookies từ browser của bạn và dùng trong yt-dlp, bạn "giả" là người dùng đã đăng nhập.

## Cách 1: Sử dụng Extension (KHUYẾN NGHỊ)

### Chrome/Edge:

1. Install extension: [Get cookies.txt LOCALLY](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc)
2. Vào https://www.youtube.com (đảm bảo đã đăng nhập)
3. Click extension icon → Export cookies
4. Save file `youtube.com_cookies.txt`

### Firefox:

1. Install addon: [cookies.txt](https://addons.mozilla.org/en-US/firefox/addon/cookies-txt/)
2. Vào https://www.youtube.com (đảm bảo đã đăng nhập)
3. Click addon icon → Export cookies
4. Save file `youtube.com_cookies.txt`

## Cách 2: Manual Export (Developer Tools)

1. Vào https://www.youtube.com
2. Press F12 → Application tab → Cookies → https://www.youtube.com
3. Copy cookies quan trọng: `__Secure-1PSID`, `__Secure-3PSID`, `VISITOR_INFO1_LIVE`

## Sử dụng cookies trong project

### Option A: Upload cookies file (Local dev)

1. Đặt file `youtube_cookies.txt` vào folder gốc project
2. Update config: `yt_dlp_cookies_path: "./youtube_cookies.txt"`

### Option B: Environment Variable (Render - KHUYẾN NGHỊ)

1. Convert cookies file sang base64:
   ```bash
   # Windows PowerShell
   [Convert]::ToBase64String([IO.File]::ReadAllBytes("youtube_cookies.txt"))
   
   # Linux/Mac
   base64 -w 0 youtube_cookies.txt
   ```

2. Copy output và set environment variable trên Render:
   ```
   YOUTUBE_COOKIES_BASE64=<paste_base64_here>
   ```

3. Application sẽ tự động decode và sử dụng

## Lưu ý quan trọng

⚠️ **KHÔNG commit cookies lên Git** - Chứa thông tin đăng nhập của bạn!

✅ Cookies đã được thêm vào `.gitignore`:
```
*.txt
*_cookies.txt
youtube_cookies.txt
```

🔄 **Refresh cookies định kỳ:** Cookies có expiry, nên refresh mỗi 1-2 tháng

🔒 **Bảo mật:** Chỉ export cookies từ accounts không quan trọng hoặc test account

