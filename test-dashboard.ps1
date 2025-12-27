# Test script to generate traffic and watch the dashboard update

Write-Host "🚦 SignalFence Dashboard Test" -ForegroundColor Cyan
Write-Host "Dashboard: http://localhost:8080/dashboard" -ForegroundColor Green
Write-Host ""
Write-Host "Generating realistic traffic patterns..." -ForegroundColor Yellow
Write-Host "  - Normal users: moderate traffic" -ForegroundColor White
Write-Host "  - Bots/scrapers: aggressive traffic (will get blocked)" -ForegroundColor White
Write-Host "  - Premium users: high limits" -ForegroundColor White
Write-Host ""

# Define different user types with different behaviors
$normalUsers = @("user-alice", "user-bob", "user-charlie")
$aggressiveBots = @("bot-scraper", "bot-crawler", "suspicious-ip")
$premiumUsers = @("premium-api-key", "enterprise-client")

Write-Host "Starting traffic generation..." -ForegroundColor Cyan
Write-Host ""

for ($round = 1; $round -le 15; $round++) {
    Write-Host "Round $round/15" -ForegroundColor Cyan
    
    # Normal users: 5-15 requests (should mostly succeed)
    foreach ($client in $normalUsers) {
        $numRequests = Get-Random -Minimum 5 -Maximum 15
        $allowed = 0
        $blocked = 0
        
        for ($i = 1; $i -le $numRequests; $i++) {
            try {
                $response = Invoke-WebRequest -Method POST `
                    -Uri "http://localhost:8080/check" `
                    -ContentType "application/json" `
                    -Body "{`"client_id`":`"$client`"}" `
                    -UseBasicParsing `
                    -ErrorAction SilentlyContinue
                $allowed++
            } catch {
                $blocked++
            }
        }
        Write-Host "  👤 $client : $allowed allowed, $blocked blocked" -ForegroundColor Green
    }
    
    # Aggressive bots: 30-50 requests with low limits (will get blocked)
    foreach ($client in $aggressiveBots) {
        $numRequests = Get-Random -Minimum 30 -Maximum 50
        $allowed = 0
        $blocked = 0
        
        for ($i = 1; $i -le $numRequests; $i++) {
            try {
                # Low limits for bots: 10 capacity, 2/sec refill
                $body = @{
                    client_id = $client
                    capacity = 10
                    refill_per_sec = 2
                } | ConvertTo-Json
                
                $response = Invoke-WebRequest -Method POST `
                    -Uri "http://localhost:8080/check" `
                    -ContentType "application/json" `
                    -Body $body `
                    -UseBasicParsing `
                    -ErrorAction SilentlyContinue
                $allowed++
            } catch {
                $blocked++
            }
        }
        Write-Host "  🤖 $client : $allowed allowed, $blocked blocked" -ForegroundColor Red
    }
    
    # Premium users: 20-30 requests with high limits (should succeed)
    foreach ($client in $premiumUsers) {
        $numRequests = Get-Random -Minimum 20 -Maximum 30
        $allowed = 0
        $blocked = 0
        
        for ($i = 1; $i -le $numRequests; $i++) {
            try {
                # High limits for premium: 200 capacity, 50/sec refill
                $body = @{
                    client_id = $client
                    capacity = 200
                    refill_per_sec = 50
                } | ConvertTo-Json
                
                $response = Invoke-WebRequest -Method POST `
                    -Uri "http://localhost:8080/check" `
                    -ContentType "application/json" `
                    -Body $body `
                    -UseBasicParsing `
                    -ErrorAction SilentlyContinue
                $allowed++
            } catch {
                $blocked++
            }
        }
        Write-Host "  💎 $client : $allowed allowed, $blocked blocked" -ForegroundColor Cyan
    }
    
    Write-Host ""
    Start-Sleep -Milliseconds 500
}

Write-Host ""
Write-Host "Test complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Check the dashboard to see:" -ForegroundColor Yellow
Write-Host "  Bots with high block rates" -ForegroundColor Red
Write-Host "  Normal users with low block rates" -ForegroundColor Green
Write-Host "  Premium users with minimal blocks" -ForegroundColor Cyan
Write-Host ""
Write-Host "Dashboard: http://localhost:8080/dashboard" -ForegroundColor Green
