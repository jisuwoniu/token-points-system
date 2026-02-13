const API_BASE_URL = '/api';

let pointsChart = null;

document.addEventListener('DOMContentLoaded', function() {
    loadDashboardData();
    initPointsChart();
    loadRecentTransactions();
    
    setInterval(loadDashboardData, 30000);
});

async function loadDashboardData() {
    try {
        const response = await axios.get(`${API_BASE_URL}/stats`);
        const data = response.data;
        
        document.getElementById('totalUsers').textContent = formatNumber(data.totalUsers || 0);
        document.getElementById('totalPoints').textContent = formatNumber(data.totalPoints || 0);
        document.getElementById('totalTx').textContent = formatNumber(data.totalTransactions || 0);
        document.getElementById('sepoliaBlock').textContent = formatNumber(data.sepoliaBlock || 0);
        document.getElementById('baseBlock').textContent = formatNumber(data.baseBlock || 0);
    } catch (error) {
        console.error('Failed to load dashboard data:', error);
        showNotification('Failed to load data', 'danger');
    }
}

function initPointsChart() {
    const ctx = document.getElementById('pointsChart').getContext('2d');
    
    pointsChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Points Earned',
                data: [],
                borderColor: '#667eea',
                backgroundColor: 'rgba(102, 126, 234, 0.1)',
                tension: 0.4,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    grid: {
                        color: 'rgba(0,0,0,0.05)'
                    }
                },
                x: {
                    grid: {
                        display: false
                    }
                }
            }
        }
    });
    
    loadChartData();
}

async function loadChartData() {
    try {
        const response = await axios.get(`${API_BASE_URL}/points/history`);
        const data = response.data;
        
        pointsChart.data.labels = data.labels || [];
        pointsChart.data.datasets[0].data = data.values || [];
        pointsChart.update();
    } catch (error) {
        console.error('Failed to load chart data:', error);
    }
}

async function loadRecentTransactions() {
    try {
        const response = await axios.get(`${API_BASE_URL}/transactions/recent`);
        const transactions = response.data;
        
        const tbody = document.getElementById('recentTxTable');
        tbody.innerHTML = '';
        
        if (transactions && transactions.length > 0) {
            transactions.forEach(tx => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td><span class="badge bg-info">${tx.chain}</span></td>
                    <td><span class="badge bg-${getTypeColor(tx.type)}">${tx.type}</span></td>
                    <td><small>${shortenAddress(tx.from)}</small></td>
                    <td><small>${shortenAddress(tx.to)}</small></td>
                    <td>${formatNumber(tx.amount)}</td>
                    <td><small>${formatTime(tx.timestamp)}</small></td>
                `;
                tbody.appendChild(row);
            });
        } else {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">No recent transactions</td></tr>';
        }
    } catch (error) {
        console.error('Failed to load transactions:', error);
    }
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(2) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(2) + 'K';
    }
    return num.toString();
}

function shortenAddress(address) {
    if (!address) return '';
    return `${address.substring(0, 6)}...${address.substring(address.length - 4)}`;
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString();
}

function getTypeColor(type) {
    switch(type.toLowerCase()) {
        case 'mint': return 'success';
        case 'burn': return 'danger';
        case 'transfer': return 'primary';
        default: return 'secondary';
    }
}

function showNotification(message, type) {
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    alertDiv.style.cssText = 'top: 20px; right: 20px; z-index: 9999;';
    alertDiv.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
    `;
    document.body.appendChild(alertDiv);
    
    setTimeout(() => {
        alertDiv.remove();
    }, 5000);
}
