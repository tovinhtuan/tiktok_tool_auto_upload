# Script to fix expired token by exchanging authorization code
# Usage: .\fix_expired_token.ps1 -AccountId <account_id>

param(
    [Parameter(Mandatory=$true)]
    [string]$AccountId,
    
    [Parameter(Mandatory=$false)]
    [string]$ApiBaseUrl = "http://localhost:8080"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Fix Expired TikTok Token" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Get authorization URL
$clientKey = "sbawggp68sw26gl2hv"
$redirectUri = "https://tovinhtuan.github.io/tiktok-policy/callback"
$scope = "user.info.basic,video.upload"
$state = "12345"

$authUrl = "https://www.tiktok.com/v2/auth/authorize/?client_key=$clientKey&scope=$scope&response_type=code&redirect_uri=$([System.Web.HttpUtility]::UrlEncode($redirectUri))&state=$state"

Write-Host "Step 1: Get Authorization Code" -ForegroundColor Yellow
Write-Host "1. Open this URL in your browser:" -ForegroundColor White
Write-Host "   $authUrl" -ForegroundColor Green
Write-Host ""
Write-Host "2. Authorize the application" -ForegroundColor White
Write-Host "3. After authorization, you will be redirected to a callback URL" -ForegroundColor White
Write-Host "4. Copy the entire callback URL (script will extract the 'code' parameter automatically)" -ForegroundColor White
Write-Host ""
Write-Host "Example callback URL:" -ForegroundColor Gray
Write-Host "   https://tovinhtuan.github.io/tiktok-policy/callback?code=YOUR_CODE_HERE&scopes=...&state=12345" -ForegroundColor Gray
Write-Host ""

# Helper to extract code from callback URL or raw input
function Get-AuthorizationCodeFromInput {
    param([string]$input)

    if ([string]::IsNullOrWhiteSpace($input)) {
        return ""
    }

    # If user pasted full callback URL, parse query string for code parameter
    if ($input.Trim().StartsWith("http", [System.StringComparison]::OrdinalIgnoreCase)) {
        try {
            $uri = [System.Uri]$input.Trim()
            $queryParams = [System.Web.HttpUtility]::ParseQueryString($uri.Query)
            $codeFromUrl = $queryParams["code"]
            if (-not [string]::IsNullOrWhiteSpace($codeFromUrl)) {
                return $codeFromUrl
            }
        } catch {
            Write-Host "Warning: Failed to parse callback URL, falling back to raw input." -ForegroundColor Yellow
        }
    }

    return $input.Trim()
}

# Step 2: Get code from user or parse from callback URL
$callbackOrCode = Read-Host "Paste the full callback URL (or just the authorization code)"
$code = Get-AuthorizationCodeFromInput -input $callbackOrCode

if ([string]::IsNullOrEmpty($code)) {
    Write-Host "Error: Authorization code is required" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Step 2: Exchanging code for token..." -ForegroundColor Yellow
Write-Host "Using code: $code" -ForegroundColor Gray
Write-Host ""
Write-Host "Reminder: This code will be sent in the 'code' field when calling https://open.tiktokapis.com/v2/oauth/token/ (handled automatically via /api/tiktok/exchange-code)." -ForegroundColor DarkGray

# Step 3: Exchange code for token
$body = @{
    code = $code
    redirect_uri = $redirectUri
    account_id = $AccountId
} | ConvertTo-Json

$headers = @{
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri "$ApiBaseUrl/api/tiktok/exchange-code" `
        -Method POST `
        -Headers $headers `
        -Body $body `
        -ErrorAction Stop
    
    Write-Host ""
    Write-Host "Success! Token updated successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Account Details:" -ForegroundColor Cyan
    Write-Host "  Account ID: $($response.account.id)" -ForegroundColor White
    Write-Host "  YouTube Channel: $($response.account.youtube_channel_id)" -ForegroundColor White
    Write-Host "  TikTok Account: $($response.account.tiktok_account_id)" -ForegroundColor White
    Write-Host "  Has Refresh Token: $($response.has_refresh_token)" -ForegroundColor $(if ($response.has_refresh_token) { "Green" } else { "Yellow" })
    Write-Host "  Token Expires In: $($response.expires_in) seconds" -ForegroundColor White
    Write-Host ""
    
    if (-not $response.has_refresh_token) {
        Write-Host "WARNING: No refresh token received!" -ForegroundColor Yellow
        Write-Host "Token will need manual update when expired." -ForegroundColor Yellow
    } else {
        Write-Host "Token will auto-refresh when expired." -ForegroundColor Green
    }
    
} catch {
    Write-Host ""
    Write-Host "Error exchanging code:" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

