// Phase-based Desktop Renderer
// Phases: login → menu → desktop
//
// IPC from main:
//   init-server-url  — backend URL to use for auth calls
//   show-menu        — login succeeded; menu URL ready
//   show-desktop     — menu done; reveal HUD + load frontend
//
// IPC to main:
//   login-success    — user authenticated
//   menu-complete    — menu animation finished (forwarded from menu iframe postMessage)
//   minimize-window  — user clicked minimize
//   close-window     — user clicked close

const os = require('os');
const { ipcRenderer } = require('electron');

// ─── State ────────────────────────────────────────────────────────────────────

let serverUrl = 'http://localhost:8090';  // updated by init-server-url

// ─── DOM refs ─────────────────────────────────────────────────────────────────

const loginOverlay  = document.getElementById('login-overlay');
const menuOverlay   = document.getElementById('menu-overlay');
const menuIframe    = document.getElementById('menu-iframe');
const hudContainer  = document.getElementById('hud-container');
const contentIframe = document.getElementById('content-iframe');

// Login panel
const loginPanel    = document.getElementById('login-panel');
const loginForm     = document.getElementById('login-form');
const loginUsername = document.getElementById('login-username');
const loginPassword = document.getElementById('login-password');
const loginToken    = document.getElementById('login-token');
const loginError    = document.getElementById('login-error');
const loginBtn      = document.getElementById('login-btn');
const loginBtnText  = document.getElementById('login-btn-text');
const credsFields   = document.getElementById('creds-fields');
const tokenFields   = document.getElementById('token-fields');
const tabCreds      = document.getElementById('tab-credentials');
const tabToken      = document.getElementById('tab-token');

// Register panel
const registerPanel     = document.getElementById('register-panel');
const registerForm      = document.getElementById('register-form');
const regUsername       = document.getElementById('reg-username');
const regEmail          = document.getElementById('reg-email');
const regRole           = document.getElementById('reg-role');
const regPassword       = document.getElementById('reg-password');
const regConfirm        = document.getElementById('reg-confirm');
const registerError     = document.getElementById('register-error');
const registerSuccess   = document.getElementById('register-success');
const registerBtn       = document.getElementById('register-btn');
const registerBtnText   = document.getElementById('register-btn-text');
const showRegisterBtn   = document.getElementById('show-register-btn');
const cancelRegisterBtn = document.getElementById('cancel-register-btn');

// ─── Login mode state ─────────────────────────────────────────────────────────

let loginMode = 'credentials'; // 'credentials' | 'token'

function switchMode(mode) {
    loginMode = mode;
    if (mode === 'credentials') {
        credsFields.style.display = 'block';
        tokenFields.style.display  = 'none';
        tabCreds.classList.add('active');
        tabToken.classList.remove('active');
    } else {
        credsFields.style.display = 'none';
        tokenFields.style.display  = 'block';
        tabToken.classList.add('active');
        tabCreds.classList.remove('active');
    }
    clearLoginError();
}

tabCreds.addEventListener('click', () => switchMode('credentials'));
tabToken.addEventListener('click', () => switchMode('token'));

// Testnet token quick-fill buttons
document.querySelectorAll('.testnet-btn').forEach((btn) => {
    btn.addEventListener('click', () => {
        loginToken.value = btn.dataset.token;
        clearLoginError();
    });
});

// ─── Phase transitions ────────────────────────────────────────────────────────

function showLoginError(msg) {
    loginError.textContent  = msg;
    loginError.style.display = 'block';
}

function clearLoginError() {
    loginError.textContent  = '';
    loginError.style.display = 'none';
}

function setLoginBusy(busy) {
    loginBtn.disabled       = busy;
    loginBtnText.textContent = busy ? 'AUTHENTICATING...' : 'AUTHENTICATE';
}

// Alias kept for compatibility with the section below
const showError  = showLoginError;
const clearError = clearLoginError;

// Login → Menu: cross-fade
function transitionToMenu(menuUrl) {
    menuIframe.src = menuUrl;
    menuOverlay.style.display = 'flex';
    menuOverlay.style.opacity = '0';

    // Fade login out, menu in simultaneously
    loginOverlay.style.transition = 'opacity 0.6s ease';
    menuOverlay.style.transition  = 'opacity 0.6s ease';

    requestAnimationFrame(() => {
        loginOverlay.style.opacity = '0';
        menuOverlay.style.opacity  = '1';
    });

    setTimeout(() => {
        loginOverlay.style.display = 'none';
    }, 700);
}

// Menu → Desktop: fade menu out, fade HUD in
function transitionToDesktop(frontendUrl) {
    hudContainer.style.display = 'block';
    hudContainer.style.opacity  = '0';
    hudContainer.style.transition = 'opacity 0.8s ease';

    menuOverlay.style.transition = 'opacity 0.8s ease';

    requestAnimationFrame(() => {
        menuOverlay.style.opacity  = '0';
        hudContainer.style.opacity = '1';
    });

    setTimeout(() => {
        menuOverlay.style.display = 'none';
        // Load the frontend after the HUD is visible
        contentIframe.src = frontendUrl;
    }, 900);
}

// ─── IPC from main process ────────────────────────────────────────────────────

// Called right after the window loads — gives us the backend URL for login
ipcRenderer.on('init-server-url', (_event, url) => {
    serverUrl = url;
});

// Main process confirmed login + menu server is ready
ipcRenderer.on('show-menu', (_event, { menuUrl }) => {
    transitionToMenu(menuUrl);
});

// Menu animation done — reveal the HUD desktop frame
ipcRenderer.on('show-desktop', (_event, { frontendUrl }) => {
    transitionToDesktop(frontendUrl);
});

// Frontend connection status updates (used in desktop phase)
ipcRenderer.on('frontend-status', (_event, status) => {
    const el = document.getElementById('frontend-status');
    if (el) {
        el.textContent = status;
        el.className = 'value ' + (status === 'CONNECTED' ? 'active' : '');
    }
});

// ─── postMessage from menu iframe ─────────────────────────────────────────────

window.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'menu-complete') {
        ipcRenderer.send('menu-complete');
    }
});

// ─── Login form ───────────────────────────────────────────────────────────────

loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearLoginError();
    setLoginBusy(true);

    try {
        let token = null;
        let role  = '';

        if (loginMode === 'credentials') {
            const username = loginUsername.value.trim();
            const password = loginPassword.value;
            if (!username || !password) {
                showLoginError('Username and password are required.');
                setLoginBusy(false);
                return;
            }

            const res = await fetch(`${serverUrl}/api/auth/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
            });
            const body = await res.json().catch(() => ({}));
            if (!res.ok) {
                showLoginError(body.error || body.message || `Authentication failed (${res.status})`);
                setLoginBusy(false);
                return;
            }
            token = body.token;
            role  = body.role || '';

        } else {
            // Token mode
            const rawToken = loginToken.value.trim();
            if (!rawToken) {
                showLoginError('Please enter an access token.');
                setLoginBusy(false);
                return;
            }

            // Testnet tokens are validated locally — no network call needed
            // (mirrors auth-context.tsx validateToken behaviour)
            const testnetMap = {
                'testnet-admin-123':    { role: 'admin',     user: 'testnet-admin' },
                'testnet-validator-456':{ role: 'validator', user: 'testnet-validator' },
                'testnet-observer-789': { role: 'observer',  user: 'testnet-observer' },
            };

            if (testnetMap[rawToken]) {
                token = rawToken;
                role  = testnetMap[rawToken].role;
            } else {
                // For JWT tokens, check expiry client-side before hitting the backend
                if (rawToken.startsWith('ey')) {
                    try {
                        const payload = JSON.parse(atob(rawToken.split('.')[1]));
                        if (payload.exp && payload.exp * 1000 < Date.now()) {
                            showLoginError('Token has expired.');
                            setLoginBusy(false);
                            return;
                        }
                    } catch { /* malformed — let backend reject it */ }
                }

                // Validate unknown tokens via /api/auth/me
                const res = await fetch(`${serverUrl}/api/auth/me`, {
                    headers: { 'Authorization': `Bearer ${rawToken}` },
                });
                const body = await res.json().catch(() => ({}));
                if (!res.ok) {
                    showLoginError(body.error || body.message || 'Invalid or expired token.');
                    setLoginBusy(false);
                    return;
                }
                token = rawToken;
                role  = body.role || '';
            }
        }

        // Persist so the frontend iframe can pick it up
        if (token) {
            localStorage.setItem('knirv_auth_token', token);
            localStorage.setItem('knirv_auth_role',  role);
        }

        // Tell main — it starts the menu server and replies with show-menu
        ipcRenderer.send('login-success');

    } catch {
        showLoginError('Cannot reach the KNIRV server. Ensure the server is running.');
        setLoginBusy(false);
    }
});

// ─── Register panel ───────────────────────────────────────────────────────────

showRegisterBtn.addEventListener('click', () => {
    loginPanel.style.display    = 'none';
    registerPanel.style.display = 'block';
    clearRegisterState();
});

cancelRegisterBtn.addEventListener('click', () => {
    registerPanel.style.display = 'none';
    loginPanel.style.display    = 'block';
});

function clearRegisterState() {
    regUsername.value       = '';
    regEmail.value          = '';
    regRole.value           = 'observer';
    regPassword.value       = '';
    regConfirm.value        = '';
    registerError.textContent  = '';
    registerError.style.display   = 'none';
    registerSuccess.textContent   = '';
    registerSuccess.style.display = 'none';
}

function setRegisterBusy(busy) {
    registerBtn.disabled        = busy;
    registerBtnText.textContent = busy ? 'REGISTERING...' : 'REGISTER';
}

registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    registerError.style.display   = 'none';
    registerSuccess.style.display = 'none';

    const username = regUsername.value.trim();
    const email    = regEmail.value.trim();
    const role     = regRole.value;
    const password = regPassword.value;
    const confirm  = regConfirm.value;

    if (!username || !email || !password) {
        registerError.textContent  = 'Please fill in all required fields.';
        registerError.style.display = 'block';
        return;
    }
    if (password !== confirm) {
        registerError.textContent  = 'Passwords do not match.';
        registerError.style.display = 'block';
        return;
    }
    if (password.length < 8) {
        registerError.textContent  = 'Password must be at least 8 characters.';
        registerError.style.display = 'block';
        return;
    }

    setRegisterBusy(true);

    try {
        const res = await fetch(`${serverUrl}/api/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username,
                email,
                password,
                first_name: username,
                last_name:  username,
                role,
            }),
        });
        const body = await res.json().catch(() => ({}));

        if (res.ok) {
            registerSuccess.textContent   = 'Registration successful! You can now log in.';
            registerSuccess.style.display = 'block';
            setTimeout(() => {
                registerPanel.style.display = 'none';
                loginPanel.style.display    = 'block';
                clearRegisterState();
            }, 2500);
        } else {
            registerError.textContent  = body.error || body.message || 'Registration failed. Please try again.';
            registerError.style.display = 'block';
        }
    } catch {
        registerError.textContent  = 'Cannot reach the KNIRV server.';
        registerError.style.display = 'block';
    } finally {
        setRegisterBusy(false);
    }
});

// ─── HUD — window controls ────────────────────────────────────────────────────

document.getElementById('minimize-btn').addEventListener('click', () => {
    ipcRenderer.send('minimize-window');
});

document.getElementById('close-btn').addEventListener('click', () => {
    ipcRenderer.send('close-window');
});

// ─── HUD — system monitoring ─────────────────────────────────────────────────

const canvas = document.getElementById('performance-chart');
const ctx    = canvas.getContext('2d');
let performanceData = new Array(50).fill(0);

function updateTime() {
    const el = document.getElementById('current-time');
    if (el) el.textContent = new Date().toLocaleTimeString('en-US', { hour12: false });
}

function updateUptime() {
    const uptime  = os.uptime();
    const hours   = Math.floor(uptime / 3600);
    const minutes = Math.floor((uptime % 3600) / 60);
    const el      = document.getElementById('uptime');
    if (el) el.textContent = `${hours}h ${minutes}m`;
}

let previousCpuInfo = os.cpus();
function getCpuUsage() {
    const cpus = os.cpus();
    let totalIdle = 0, totalTick = 0;
    cpus.forEach((cpu, i) => {
        const prev = previousCpuInfo[i];
        for (const type in cpu.times) {
            totalTick += cpu.times[type] - (prev ? prev.times[type] : 0);
        }
        totalIdle += cpu.times.idle - (prev ? prev.times.idle : 0);
    });
    previousCpuInfo = cpus;
    const idle  = totalIdle / cpus.length;
    const total = totalTick / cpus.length;
    return Math.max(0, Math.min(100, 100 - ~~(100 * idle / total)));
}

function getMemoryUsage() {
    const total = os.totalmem();
    return Math.round(((total - os.freemem()) / total) * 100);
}

function drawPerformanceChart() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.strokeStyle = 'rgba(72, 136, 255, 0.1)';
    ctx.lineWidth = 1;
    for (let i = 0; i < 5; i++) {
        const y = (canvas.height / 4) * i;
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(canvas.width, y); ctx.stroke();
    }
    ctx.strokeStyle = 'rgba(72, 136, 255, 0.8)';
    ctx.lineWidth = 2;
    ctx.shadowBlur  = 10;
    ctx.shadowColor = 'rgba(72, 136, 255, 0.5)';
    ctx.beginPath();
    const spacing = canvas.width / performanceData.length;
    performanceData.forEach((v, i) => {
        const x = i * spacing;
        const y = canvas.height - (v / 100) * canvas.height;
        i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
    });
    ctx.stroke();
    ctx.shadowBlur = 0;
}

function updateMetrics() {
    const cpu = getCpuUsage();
    const mem = getMemoryUsage();

    const setEl = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val; };
    const setStyle = (id, prop, val) => { const el = document.getElementById(id); if (el) el.style[prop] = val; };

    setEl('cpu-usage',    `${cpu}%`);
    setEl('memory-usage', `${mem}%`);

    performanceData.shift();
    performanceData.push(cpu);
    drawPerformanceChart();

    const net  = Math.min(100, Math.max(0, 45 + Math.random() * 20 - 10));
    const disk = Math.min(100, Math.max(0, 60 + Math.random() * 20 - 10));
    setStyle('network-bar', 'width', `${net}%`);
    setEl('network-value', `${Math.round(net)}%`);
    setStyle('disk-bar', 'width', `${disk}%`);
    setEl('disk-value', `${Math.round(disk)}%`);

    setEl('net-rx',    `${(Math.random() * 1000).toFixed(0)} KB/s`);
    setEl('net-tx',    `${(Math.random() * 500).toFixed(0)} KB/s`);
    setEl('disk-read', `${(Math.random() * 50).toFixed(1)} MB/s`);
    setEl('disk-write',`${(Math.random() * 30).toFixed(1)} MB/s`);

    const procs = 200 + Math.floor(Math.random() * 100);
    setStyle('process-bar', 'width', `${(procs / 400) * 100}%`);
    setEl('process-value', `${procs}`);
    setEl('threads', `${Math.floor(procs * 2.5)}`);
    setEl('temp',    `${40 + Math.floor(Math.random() * 10)}°C`);
    setEl('fan',     `${2000 + Math.floor(Math.random() * 1000)} RPM`);
}

function initSystemInfo() {
    const osEl   = document.getElementById('os-info');
    const archEl = document.getElementById('arch-info');
    if (osEl)   osEl.textContent   = `${os.type()} ${os.release()}`;
    if (archEl) archEl.textContent = os.arch();
}

// Start HUD monitoring once (metrics only matter in the desktop phase,
// but we can run them in the background from the start — low overhead)
initSystemInfo();
updateTime();
updateUptime();
updateMetrics();
setInterval(updateTime,    1000);
setInterval(updateUptime, 60000);
setInterval(updateMetrics, 2000);
drawPerformanceChart();

console.log('KNIRV Desktop renderer initialised — waiting for login.');
