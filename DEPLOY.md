# Hướng dẫn Deploy lên Render

Hướng dẫn này giúp bạn deploy ứng dụng Auto Upload TikTok lên Render sử dụng Docker.

## Yêu cầu

- Tài khoản Render (https://render.com)
- GitHub/GitLab repository chứa code của bạn
- API Keys:
  - YouTube Data API v3 key
  - TikTok Open API key và secret

## Cách 1: Deploy bằng Render Blueprint (Khuyến nghị)

1. **Push code lên repository:**
   ```bash
   git add .
   git commit -m "Add Docker support for Render"
   git push
   ```

2. **Truy cập Render Dashboard:**
   - Vào https://dashboard.render.com
   - Click "New +" → "Blueprint"

3. **Connect Repository:**
   - Chọn repository của bạn
   - Render sẽ tự động detect file `render.yaml`

4. **Cấu hình Environment Variables:**
   Trong Render Dashboard, thêm các biến môi trường sau:
   
   **Bắt buộc:**
   - `YOUTUBE_API_KEY`: YouTube Data API v3 key
   - `TIKTOK_API_KEY`: TikTok Open API key
   - `TIKTOK_API_SECRET`: TikTok Open API secret
   
   **Tùy chọn:**
   - `PORT`: Port cho server (Render tự động set, không cần chỉnh)
   - `TIKTOK_REDIRECT_URI`: Callback URL cho OAuth (sau khi deploy, set thành `https://your-service.onrender.com/api/tiktok/callback`)
   - `CRON_SCHEDULE`: Lịch chạy cron (mặc định: `* * * * * *` - quét YouTube mỗi giây)
   - Các biến khác xem trong `render.yaml`

5. **Deploy:**
   - Click "Apply" để bắt đầu deploy
   - Render sẽ tự động build Docker image và deploy

## Cách 2: Deploy thủ công bằng Web Service

1. **Truy cập Render Dashboard:**
   - Vào https://dashboard.render.com
   - Click "New +" → "Web Service"

2. **Connect Repository:**
   - Chọn repository của bạn

3. **Cấu hình:**
   - **Name:** auto-upload-tiktok (hoặc tên bạn muốn)
   - **Runtime:** Docker
   - **Dockerfile Path:** `./Dockerfile`
   - **Docker Context:** `.`
   - **Start Command:** (để trống, Dockerfile đã có ENTRYPOINT)
   - **Plan:** Starter (hoặc Standard/Pro tùy nhu cầu)

4. **Environment Variables:**
   Thêm các biến môi trường tương tự như Cách 1

5. **Advanced Settings:**
   - **Health Check Path:** `/api/health` (tùy chọn)
   - **Auto-Deploy:** Yes (tự động deploy khi có commit mới)

6. **Deploy:**
   - Click "Create Web Service"
   - Chờ build và deploy hoàn tất

## Sau khi Deploy

1. **Kiểm tra Health:**
   - Vào URL service của bạn: `https://your-service.onrender.com/api/health`
   - Nếu trả về `{"status":"ok"}`, service đã chạy thành công

2. **Cập nhật TikTok Redirect URI:**
   - Vào Render Dashboard → Environment
   - Set `TIKTOK_REDIRECT_URI` = `https://your-service.onrender.com/api/tiktok/callback`
   - Restart service

3. **Truy cập Web UI:**
   - Vào `https://your-service.onrender.com/`
   - Web UI để quản lý accounts và videos

## Lưu ý quan trọng

### Persistent Storage
- Render **không lưu trữ dữ liệu** sau khi service restart
- SQLite database và downloaded files sẽ bị mất
- **Giải pháp:**
  - Sử dụng Render Disk (paid plan) cho persistent storage
  - Hoặc migrate sang PostgreSQL/MySQL (cần modify code)
  - Hoặc sử dụng external storage (S3, etc.)

### Ephemeral Storage
- Downloads và logs chỉ tồn tại trong container
- Cần cấu hình external storage cho production

### Build Time
- Build có thể mất 5-10 phút (do cần install Python, yt-dlp, ffmpeg)
- Đảm bảo timeout của Render đủ lớn

### Resource Limits
- Starter plan: 512MB RAM, có thể không đủ cho video processing
- Khuyến nghị: Standard plan trở lên cho production

## Troubleshooting

### Build fails
- Kiểm tra Dockerfile có lỗi syntax không
- Xem build logs trong Render Dashboard

### Service không start
- Kiểm tra logs trong Render Dashboard
- Đảm bảo các environment variables bắt buộc đã được set

### Health check fails
- Kiểm tra PORT environment variable
- Đảm bảo `/api/health` endpoint hoạt động

### Database errors
- Render không persist data, cần restart service để tạo database mới
- Hoặc migrate sang external database

## Environment Variables Reference

Xem file `render.yaml` để biết danh sách đầy đủ các environment variables có thể cấu hình.
