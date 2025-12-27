package main

import (
	"net/http"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SignalFence Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            color: white;
            margin-bottom: 30px;
        }
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .header p {
            opacity: 0.9;
            font-size: 1.1em;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            border-radius: 12px;
            padding: 25px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        .stat-card:hover {
            transform: translateY(-5px);
        }
        .stat-label {
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 10px;
        }
        .stat-value {
            font-size: 2.5em;
            font-weight: bold;
            color: #333;
        }
        .stat-value.success { color: #10b981; }
        .stat-value.danger { color: #ef4444; }
        .stat-value.info { color: #3b82f6; }
        .stat-value.warning { color: #f59e0b; }
        .stat-sublabel {
            margin-top: 8px;
            font-size: 0.9em;
            color: #666;
            font-weight: normal;
        }
        .table-card {
            background: white;
            border-radius: 12px;
            padding: 25px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .table-card h2 {
            margin-bottom: 20px;
            color: #333;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th {
            text-align: left;
            padding: 12px;
            background: #f3f4f6;
            color: #666;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.85em;
            letter-spacing: 0.5px;
        }
        td {
            padding: 12px;
            border-bottom: 1px solid #e5e7eb;
        }
        tr:last-child td {
            border-bottom: none;
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
        }
        .badge.success {
            background: #d1fae5;
            color: #065f46;
        }
        .badge.danger {
            background: #fee2e2;
            color: #991b1b;
        }
        .refresh-indicator {
            position: fixed;
            top: 20px;
            right: 20px;
            background: white;
            padding: 10px 20px;
            border-radius: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            font-size: 0.9em;
            color: #666;
        }
        .refresh-indicator.active {
            background: #10b981;
            color: white;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        .loading {
            animation: pulse 1.5s ease-in-out infinite;
        }
    </style>
</head>
<body>
    <div class="refresh-indicator" id="refreshIndicator">
        Auto-refresh: <span id="countdown">2</span>s
    </div>

    <div class="container">
        <div class="header">
            <h1>🚦 SignalFence</h1>
            <p>Real-time Rate Limiting Dashboard</p>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Total Requests</div>
                <div class="stat-value info" id="totalRequests">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Allowed</div>
                <div class="stat-value success" id="allowedRequests">0</div>
                <div class="stat-sublabel" id="successRate">0% success rate</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Blocked</div>
                <div class="stat-value danger" id="blockedRequests">0</div>
                <div class="stat-sublabel" id="blockRate">0% block rate</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Unique Clients</div>
                <div class="stat-value warning" id="uniqueClients">0</div>
            </div>
        </div>

        <div class="table-card">
            <h2>Top Clients</h2>
            <table>
                <thead>
                    <tr>
                        <th>Client ID</th>
                        <th>Total</th>
                        <th>Allowed</th>
                        <th>Blocked</th>
                        <th>Block Rate</th>
                        <th>Last Seen</th>
                    </tr>
                </thead>
                <tbody id="topClientsTable">
                    <tr>
                        <td colspan="6" style="text-align: center; color: #999;">
                            Loading...
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>

    <script>
        let countdown = 2;
        let countdownInterval;

        async function fetchMetrics() {
            try {
                const response = await fetch('/metrics');
                const data = await response.json();
                updateDashboard(data);
            } catch (error) {
                console.error('Failed to fetch metrics:', error);
            }
        }

        function updateDashboard(data) {
            // Update stats
            document.getElementById('totalRequests').textContent = 
                data.total_requests.toLocaleString();
            document.getElementById('allowedRequests').textContent = 
                data.allowed_requests.toLocaleString();
            document.getElementById('blockedRequests').textContent = 
                data.blocked_requests.toLocaleString();
            document.getElementById('uniqueClients').textContent = 
                data.unique_clients.toLocaleString();

            // Calculate and display rates
            if (data.total_requests > 0) {
                const successRate = ((data.allowed_requests / data.total_requests) * 100).toFixed(1);
                const blockRate = ((data.blocked_requests / data.total_requests) * 100).toFixed(1);
                document.getElementById('successRate').textContent = successRate + '% success rate';
                document.getElementById('blockRate').textContent = blockRate + '% block rate';
            } else {
                document.getElementById('successRate').textContent = '0% success rate';
                document.getElementById('blockRate').textContent = '0% block rate';
            }

            // Update top clients table
            const tbody = document.getElementById('topClientsTable');
            if (data.top_clients && data.top_clients.length > 0) {
                tbody.innerHTML = data.top_clients.map(client => {
                    const blockRate = ((client.BlockedRequests / client.TotalRequests) * 100).toFixed(1);
                    const lastSeen = new Date(client.LastRequestAt).toLocaleTimeString();
                    
                    return ` + "`" + `
                        <tr>
                            <td><strong>${client.ClientID}</strong></td>
                            <td>${client.TotalRequests.toLocaleString()}</td>
                            <td><span class="badge success">${client.AllowedRequests}</span></td>
                            <td><span class="badge danger">${client.BlockedRequests}</span></td>
                            <td>${blockRate}%</td>
                            <td>${lastSeen}</td>
                        </tr>
                    ` + "`" + `;
                }).join('');
            } else {
                tbody.innerHTML = ` + "`" + `
                    <tr>
                        <td colspan="6" style="text-align: center; color: #999;">
                            No requests yet
                        </td>
                    </tr>
                ` + "`" + `;
            }
        }

        function startCountdown() {
            countdown = 2;
            document.getElementById('countdown').textContent = countdown;
            
            if (countdownInterval) clearInterval(countdownInterval);
            
            countdownInterval = setInterval(() => {
                countdown--;
                document.getElementById('countdown').textContent = countdown;
                
                if (countdown <= 0) {
                    countdown = 2;
                }
            }, 1000);
        }

        // Initial fetch
        fetchMetrics();
        startCountdown();

        // Auto-refresh every 2 seconds
        setInterval(() => {
            fetchMetrics();
            startCountdown();
        }, 2000);
    </script>
</body>
</html>`
