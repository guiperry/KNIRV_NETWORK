// System monitoring and UI updates
const os = require('os');
const { ipcRenderer } = require('electron');

const iframe = document.getElementById('content-iframe');

ipcRenderer.on('set-frontend-url', (event, url) => {
    console.log('Loading frontend URL:', url);
    iframe.src = url;
});

ipcRenderer.on('frontend-status', (event, status) => {
    const statusEl = document.getElementById('frontend-status');
    if (statusEl) {
        statusEl.textContent = status;
        statusEl.className = 'value ' + (status === 'CONNECTED' ? 'active' : '');
    }
});

// Initialize performance chart
const canvas = document.getElementById('performance-chart');
const ctx = canvas.getContext('2d');
let performanceData = new Array(50).fill(0);

// Update time
function updateTime() {
    const now = new Date();
    const timeString = now.toLocaleTimeString('en-US', { hour12: false });
    document.getElementById('current-time').textContent = timeString;
}

// Update system uptime
function updateUptime() {
    const uptime = os.uptime();
    const hours = Math.floor(uptime / 3600);
    const minutes = Math.floor((uptime % 3600) / 60);
    document.getElementById('uptime').textContent = `${hours}h ${minutes}m`;
}

// Get CPU usage (approximation)
let previousCpuInfo = os.cpus();
function getCpuUsage() {
    const cpus = os.cpus();
    let totalIdle = 0;
    let totalTick = 0;

    cpus.forEach((cpu, i) => {
        const prevCpu = previousCpuInfo[i];
        
        for (let type in cpu.times) {
            totalTick += cpu.times[type] - (prevCpu ? prevCpu.times[type] : 0);
        }
        
        totalIdle += cpu.times.idle - (prevCpu ? prevCpu.times.idle : 0);
    });

    previousCpuInfo = cpus;
    const idle = totalIdle / cpus.length;
    const total = totalTick / cpus.length;
    const usage = 100 - ~~(100 * idle / total);
    
    return Math.max(0, Math.min(100, usage));
}

// Get memory usage
function getMemoryUsage() {
    const totalMem = os.totalmem();
    const freeMem = os.freemem();
    const usedMem = totalMem - freeMem;
    return Math.round((usedMem / totalMem) * 100);
}

// Draw performance chart
function drawPerformanceChart() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    // Draw grid
    ctx.strokeStyle = 'rgba(72, 136, 255, 0.1)';
    ctx.lineWidth = 1;
    for (let i = 0; i < 5; i++) {
        const y = (canvas.height / 4) * i;
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(canvas.width, y);
        ctx.stroke();
    }
    
    // Draw performance line
    ctx.strokeStyle = 'rgba(72, 136, 255, 0.8)';
    ctx.lineWidth = 2;
    ctx.beginPath();
    
    const pointSpacing = canvas.width / performanceData.length;
    
    performanceData.forEach((value, index) => {
        const x = index * pointSpacing;
        const y = canvas.height - (value / 100) * canvas.height;
        
        if (index === 0) {
            ctx.moveTo(x, y);
        } else {
            ctx.lineTo(x, y);
        }
    });
    
    ctx.stroke();
    
    // Draw glow effect
    ctx.shadowBlur = 10;
    ctx.shadowColor = 'rgba(72, 136, 255, 0.5)';
    ctx.stroke();
    ctx.shadowBlur = 0;
}

// Update all metrics
function updateMetrics() {
    // CPU
    const cpuUsage = getCpuUsage();
    document.getElementById('cpu-usage').textContent = `${cpuUsage}%`;
    
    // Memory
    const memUsage = getMemoryUsage();
    document.getElementById('memory-usage').textContent = `${memUsage}%`;
    
    // Update performance chart data
    performanceData.shift();
    performanceData.push(cpuUsage);
    drawPerformanceChart();
    
    // Update bars (simulated data with some randomness)
    const networkUsage = Math.min(100, Math.max(0, 45 + Math.random() * 20 - 10));
    document.getElementById('network-bar').style.width = `${networkUsage}%`;
    document.getElementById('network-value').textContent = `${Math.round(networkUsage)}%`;
    
    const diskUsage = Math.min(100, Math.max(0, 60 + Math.random() * 20 - 10));
    document.getElementById('disk-bar').style.width = `${diskUsage}%`;
    document.getElementById('disk-value').textContent = `${Math.round(diskUsage)}%`;
    
    // Random network speeds
    const netRx = (Math.random() * 1000).toFixed(0);
    const netTx = (Math.random() * 500).toFixed(0);
    document.getElementById('net-rx').textContent = `${netRx} KB/s`;
    document.getElementById('net-tx').textContent = `${netTx} KB/s`;
    
    // Random disk speeds
    const diskRead = (Math.random() * 50).toFixed(1);
    const diskWrite = (Math.random() * 30).toFixed(1);
    document.getElementById('disk-read').textContent = `${diskRead} MB/s`;
    document.getElementById('disk-write').textContent = `${diskWrite} MB/s`;
    
    // Update process count
    const processes = 200 + Math.floor(Math.random() * 100);
    document.getElementById('process-bar').style.width = `${(processes / 400) * 100}%`;
    document.getElementById('process-value').textContent = `${processes}`;
    
    // Update other stats
    document.getElementById('threads').textContent = Math.floor(processes * 2.5);
    document.getElementById('temp').textContent = `${40 + Math.floor(Math.random() * 10)}°C`;
    document.getElementById('fan').textContent = `${2000 + Math.floor(Math.random() * 1000)} RPM`;
}

// Initialize system info
function initializeSystemInfo() {
    document.getElementById('os-info').textContent = `${os.type()} ${os.release()}`;
    document.getElementById('arch-info').textContent = os.arch();
}

// Control buttons
document.getElementById('minimize-btn').addEventListener('click', () => {
    ipcRenderer.send('minimize-window');
});

document.getElementById('close-btn').addEventListener('click', () => {
    ipcRenderer.send('close-window');
});

// Initialize
initializeSystemInfo();
updateTime();
updateUptime();
updateMetrics();

// Update intervals
setInterval(updateTime, 1000);
setInterval(updateUptime, 60000);
setInterval(updateMetrics, 2000);

// Initial chart draw
drawPerformanceChart();

console.log('Dashboard initialized successfully!');
