# Script to update TikTok access token for an account
# Usage: .\update_token.ps1 -AccountId <account_id> -NewToken <new_token>
# Or: .\update_token.ps1 -AccountId <account_id> (will prompt for token)

param(
    [Parameter(Mandatory=$true)]
    [string]$AccountId,
    
    [Parameter(Mandatory=$false)]
    [string]$NewToken,
    
    [Parameter(Mandatory=$false)]
    [string]$ApiBaseUrl = "http://localhost:8080"
)

# If token not provided, prompt for it
if ([string]::IsNullOrEmpty($NewToken)) {
    $NewToken = Read-Host "Enter new TikTok access token" -AsSecureString
    $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($NewToken)
    $NewToken = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
}

Write-Host "Updating token for account: $AccountId" -ForegroundColor Yellow

# Prepare the request
$body = @{
    tiktok_access_token = $NewToken
} | ConvertTo-Json

$headers = @{
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri "$ApiBaseUrl/api/accounts/$AccountId" `
        -Method PATCH `
        -Headers $headers `
        -Body $body `
        -ErrorAction Stop
    
    Write-Host "Token updated successfully!" -ForegroundColor Green
    Write-Host "Account ID: $($response.id)" -ForegroundColor Cyan
    Write-Host "YouTube Channel: $($response.youtube_channel_id)" -ForegroundColor Cyan
    Write-Host "TikTok Account: $($response.tiktok_account_id)" -ForegroundColor Cyan
    Write-Host "Is Active: $($response.is_active)" -ForegroundColor Cyan
} catch {
    Write-Host "Error updating token: $_" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

