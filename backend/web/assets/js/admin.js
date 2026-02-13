const API_BASE_URL = '/api';

document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('recalcForm').addEventListener('submit', handleRecalculate);
    loadBackups();
    loadUptime();
});

async function handleRecalculate(e) {
    e.preventDefault();
    
    const chain = document.getElementById('recalcChain').value;
    const startTime = document.getElementById('recalcStartTime').value;
    const endTime = document.getElementById('recalcEndTime').value;
    
    if (!startTime || !endTime) {
        showNotification('Please select start and end time', 'warning');
        return;
    }
    
    if (new Date(startTime) >= new Date(endTime)) {
        showNotification('End time must be after start time', 'warning');
        return;
    }
    
    if (!confirm('This will recalculate points for all users. Continue?')) {
        return;
    }
    
    const btn = e.target.querySelector('button[type="submit"]');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Processing...';
    
    try {
        const response = await axios.post(`${API_BASE_URL}/recalculate`, {
            chain: chain,
            startTime: startTime,
            endTime: endTime
        });
        
        showNotification('Recalculation started successfully!', 'success');
        addLog('info', `Points recalculation started for ${chain}`);
        
    } catch (error) {
        console.error('Recalculation failed:', error);
        showNotification('Failed to start recalculation', 'danger');
        addLog('error', `Recalculation failed: ${error.message}`);
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<i class="bi bi-calculator"></i> Start Recalculation';
    }
}

async function createBackup(chain) {
    try {
        const response = await axios.post(`${API_BASE_URL}/backup`, { chain: chain });
        showNotification(`Backup created for ${chain}`, 'success');
        addLog('info', `Backup created for ${chain}`);
        loadBackups();
    } catch (error) {
        console.error('Backup failed:', error);
        showNotification('Failed to create backup', 'danger');
        addLog('error', `Backup failed: ${error.message}`);
    }
}

async function loadBackups() {
    try {
        const response = await axios.get(`${API_BASE_URL}/backups`);
        const backups = response.data;
        
        const container = document.getElementById('backupList');
        
        if (backups && backups.length > 0) {
            container.innerHTML = backups.map(backup => `
                <div class="d-flex justify-content-between align-items-center mb-2">
                    <div>
                        <span class="badge bg-info">${backup.chain}</span>
                        <small class="text-muted ms-2">${formatTime(backup.createdAt)}</small>
                    </div>
                    <button class="btn btn-sm btn-outline-secondary" onclick="restoreBackup('${backup.id}')">
                        Restore
                    </button>
                </div>
            `).join('');
        } else {
            container.innerHTML = '<p class="text-muted small">No backups available</p>';
        }
    } catch (error) {
        console.error('Failed to load backups:', error);
    }
}

async function restoreBackup(backupId) {
    if (!confirm('This will restore data from backup. Continue?')) {
        return;
    }
    
    try {
        await axios.post(`${API_BASE_URL}/backup/restore`, { backupId: backupId });
        showNotification('Backup restored successfully', 'success');
        addLog('info', `Backup restored: ${backupId}`);
    } catch (error) {
        console.error('Restore failed:', error);
        showNotification('Failed to restore backup', 'danger');
        addLog('error', `Restore failed: ${error.message}`);
    }
}

function filterLogs(level) {
    const buttons = document.querySelectorAll('.btn-group .btn');
    buttons.forEach(btn => btn.classList.remove('active'));
    event.target.classList.add('active');
}

function addLog(level, message) {
    const logContent = document.getElementById('logContent');
    const timestamp = new Date().toISOString().replace('T', ' ').substring(0, 19);
    const colorClass = level === 'error' ? 'text-danger' : level === 'warning' ? 'text-warning' : 'text-info';
    
    const logEntry = document.createElement('div');
    logEntry.className = colorClass;
    logEntry.textContent = `[${timestamp}] ${message}`;
    
    logContent.appendChild(logEntry);
    logContent.scrollTop = logContent.scrollHeight;
}

function loadUptime() {
    const startTime = Date.now();
    setInterval(() => {
        const elapsed = Math.floor((Date.now() - startTime) / 1000);
        const hours = Math.floor(elapsed / 3600);
        const minutes = Math.floor((elapsed % 3600) / 60);
        const seconds = elapsed % 60;
        
        document.getElementById('uptime').textContent = 
            `${hours}h ${minutes}m ${seconds}s`;
    }, 1000);
}

function formatTime(timestamp) {
    if (!timestamp) return '-';
    const date = new Date(timestamp);
    return date.toLocaleString();
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
