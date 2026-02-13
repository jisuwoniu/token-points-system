const API_BASE_URL = '/api';

document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('queryForm').addEventListener('submit', handleQuery);
});

async function handleQuery(e) {
    e.preventDefault();
    
    const chain = document.getElementById('chainSelect').value;
    const address = document.getElementById('userAddress').value.trim();
    
    if (!address) {
        showNotification('Please enter a valid address', 'warning');
        return;
    }
    
    showLoading(true);
    
    try {
        const [balanceRes, pointsRes, historyRes] = await Promise.all([
            axios.get(`${API_BASE_URL}/balance/${chain}/${address}`),
            axios.get(`${API_BASE_URL}/points/${chain}/${address}`),
            axios.get(`${API_BASE_URL}/history/${chain}/${address}`)
        ]);
        
        displayBalance(balanceRes.data);
        displayPoints(pointsRes.data);
        displayHistory(historyRes.data);
        
        document.getElementById('resultSection').style.display = 'block';
        document.getElementById('historySection').style.display = 'block';
        
    } catch (error) {
        console.error('Query failed:', error);
        showNotification('Failed to query user information', 'danger');
    } finally {
        showLoading(false);
    }
}

function displayBalance(data) {
    document.getElementById('currentBalance').textContent = formatNumber(data.balance || 0);
    document.getElementById('lastUpdated').textContent = formatTime(data.updatedAt);
}

function displayPoints(data) {
    document.getElementById('totalPoints').textContent = formatNumber(data.totalPoints || 0);
    document.getElementById('lastCalculated').textContent = formatTime(data.lastCalculatedAt);
}

function displayHistory(data) {
    const tbody = document.getElementById('historyTable');
    tbody.innerHTML = '';
    
    if (data && data.length > 0) {
        data.forEach(item => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><small>${formatTime(item.timestamp)}</small></td>
                <td><span class="badge bg-${getTypeColor(item.changeType)}">${item.changeType}</span></td>
                <td>${formatNumber(item.balanceBefore)}</td>
                <td>${formatNumber(item.balanceAfter)}</td>
                <td class="${item.changeAmount.startsWith('-') ? 'text-danger' : 'text-success'}">
                    ${item.changeAmount.startsWith('-') ? '' : '+'}${formatNumber(item.changeAmount)}
                </td>
                <td>
                    <a href="#" class="text-decoration-none" onclick="copyToClipboard('${item.txHash}')">
                        <small>${shortenAddress(item.txHash)}</small>
                        <i class="bi bi-clipboard"></i>
                    </a>
                </td>
            `;
            tbody.appendChild(row);
        });
    } else {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">No history found</td></tr>';
    }
}

function formatNumber(num) {
    if (typeof num === 'string') {
        num = parseFloat(num);
    }
    if (isNaN(num)) return '0';
    
    if (num >= 1000000) {
        return (num / 1000000).toFixed(2) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(2) + 'K';
    }
    return num.toFixed(2);
}

function formatTime(timestamp) {
    if (!timestamp) return '-';
    const date = new Date(timestamp);
    return date.toLocaleString();
}

function shortenAddress(address) {
    if (!address) return '';
    return `${address.substring(0, 10)}...${address.substring(address.length - 8)}`;
}

function getTypeColor(type) {
    switch(type.toLowerCase()) {
        case 'mint': return 'success';
        case 'burn': return 'danger';
        case 'transfer': return 'primary';
        default: return 'secondary';
    }
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showNotification('Copied to clipboard!', 'success');
    }).catch(err => {
        console.error('Failed to copy:', err);
    });
}

function showLoading(show) {
    const btn = document.querySelector('#queryForm button[type="submit"]');
    if (show) {
        btn.disabled = true;
        btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Loading...';
    } else {
        btn.disabled = false;
        btn.innerHTML = '<i class="bi bi-search"></i> Query';
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
    }, 3000);
}
