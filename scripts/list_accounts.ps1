# Script to list all accounts
# Usage: .\list_accounts.ps1

param(
    [Parameter(Mandatory=$false)]
    [string]$ApiBaseUrl = "http://localhost:8080"
)

Write-Host "Fetching accounts from $ApiBaseUrl..." -ForegroundColor Yellow

try {
    $accounts = Invoke-RestMethod -Uri "$ApiBaseUrl/api/accounts" `
        -Method GET `
        -ErrorAction Stop
    
    if ($accounts.Count -eq 0) {
        Write-Host "No accounts found." -ForegroundColor Yellow
        exit 0
    }
    
    Write-Host "`nFound $($accounts.Count) account(s):`n" -ForegroundColor Green
    
    foreach ($account in $accounts) {
        Write-Host "Account ID: $($account.id)" -ForegroundColor Cyan
        Write-Host "  YouTube Channel: $($account.youtube_channel_id)" -ForegroundColor White
        Write-Host "  TikTok Account: $($account.tiktok_account_id)" -ForegroundColor White
        Write-Host "  Is Active: $($account.is_active)" -ForegroundColor White
        Write-Host "  Created: $($account.created_at)" -ForegroundColor Gray
        Write-Host ""
    }
} catch {
    Write-Host "Error fetching accounts: $_" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

