# Script to automatically find and update TikTok access token
# Usage: .\update_token_auto.ps1 -TikTokAccountId <tiktok_account_id> -NewToken <new_token>

param(
    [Parameter(Mandatory=$true)]
    [string]$TikTokAccountId,
    
    [Parameter(Mandatory=$false)]
    [string]$NewToken,
    
    [Parameter(Mandatory=$false)]
    [string]$ApiBaseUrl = "http://localhost:8080"
)

# If token not provided, prompt for it
if ([string]::IsNullOrEmpty($NewToken)) {
    $secureToken = Read-Host "Enter new TikTok access token" -AsSecureString
    $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureToken)
    $NewToken = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
}

Write-Host "Searching for account with TikTok Account ID: $TikTokAccountId" -ForegroundColor Yellow

try {
    # Get all accounts
    $accounts = Invoke-RestMethod -Uri "$ApiBaseUrl/api/accounts" `
        -Method GET `
        -ErrorAction Stop
    
    # Find account by TikTok Account ID
    $account = $accounts | Where-Object { $_.tiktok_account_id -eq $TikTokAccountId }
    
    if (-not $account) {
        Write-Host "Account not found with TikTok Account ID: $TikTokAccountId" -ForegroundColor Red
        Write-Host "Available accounts:" -ForegroundColor Yellow
        foreach ($acc in $accounts) {
            Write-Host "  - Account ID: $($acc.id), TikTok: $($acc.tiktok_account_id)" -ForegroundColor Gray
        }
        exit 1
    }
    
    Write-Host "Found account: $($account.id)" -ForegroundColor Green
    Write-Host "  YouTube Channel: $($account.youtube_channel_id)" -ForegroundColor Cyan
    Write-Host "  TikTok Account: $($account.tiktok_account_id)" -ForegroundColor Cyan
    
    # Update token
    Write-Host "`nUpdating token..." -ForegroundColor Yellow
    
    $body = @{
        tiktok_access_token = $NewToken
    } | ConvertTo-Json
    
    $headers = @{
        "Content-Type" = "application/json"
    }
    
    $response = Invoke-RestMethod -Uri "$ApiBaseUrl/api/accounts/$($account.id)" `
        -Method PATCH `
        -Headers $headers `
        -Body $body `
        -ErrorAction Stop
    
    Write-Host "`nToken updated successfully!" -ForegroundColor Green
    Write-Host "Account ID: $($response.id)" -ForegroundColor Cyan
    Write-Host "Is Active: $($response.is_active)" -ForegroundColor Cyan
    
} catch {
    Write-Host "Error: $_" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

