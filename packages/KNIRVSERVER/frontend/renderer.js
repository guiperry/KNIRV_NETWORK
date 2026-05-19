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

const electronAPI = window.electronAPI || null;
const isElectron = !!electronAPI;
const isPWA = window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone;

const fallbackAPI = {
    send: () => {},
    onInitServerUrl: (cb) => {
        // Auto-detect server URL in browser/PWA mode
        const detectedUrl = window.location.origin;
        setTimeout(() => cb(detectedUrl), 0);
    },
    onShowMenu: (cb) => {
        // In browser/PWA mode, menu is immediately available locally
        setTimeout(() => cb({ menuUrl: './menu/' }), 100);
    },
    onShowDesktop: (cb) => {
        // In browser/PWA mode, desktop loads current origin
        setTimeout(() => cb({ frontendUrl: window.location.origin }), 0);
    },
    onFrontendStatus: () => {},
    getSystemInfo: () => ({
        type: navigator.platform || 'Unknown OS',
        release: '',
        arch: navigator.userAgent || '',
    }),
    getSystemMetrics: () => ({
        uptime: Math.floor(Date.now() / 1000) - (window.performance?.timeOrigin / 1000 || 0),
        totalMem: navigator.deviceMemory ? navigator.deviceMemory * 1024 * 1024 * 1024 : 0,
        freeMem: 0,
        cpus: [],
    }),
};

const api = electronAPI || fallbackAPI;

// ─── State ────────────────────────────────────────────────────────────────────

let serverUrl     = 'http://localhost:8090';  // updated by init-server-url
let frontendUrl   = '';                        // set by show-desktop
let desktopLoaded = false;                     // true after first desktop transition
let authSession   = { token: '', role: '' };   // forwarded into the embedded frontend

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
// sectionParam is optional — appended as ?nav=<section> on first load
function transitionToDesktop(url, sectionParam) {
    frontendUrl = url;

    hudContainer.style.display   = 'block';
    hudContainer.style.opacity   = '0';
    hudContainer.style.transition = 'opacity 0.8s ease';
    menuOverlay.style.transition  = 'opacity 0.8s ease';

    // Show loading indicator
    const loadingIndicator = document.getElementById('loading-indicator');
    if (loadingIndicator) {
        loadingIndicator.style.display = 'flex';
    }

    requestAnimationFrame(() => {
        menuOverlay.style.opacity  = '0';
        hudContainer.style.opacity = '1';
    });

    setTimeout(() => {
        menuOverlay.style.display = 'none';

        if (!desktopLoaded) {
            // First load — encode the section in the URL so the app reads it on mount
            const loadUrl = sectionParam
                ? `${url}?nav=${encodeURIComponent(sectionParam)}`
                : url;
            contentIframe.src = loadUrl;
            desktopLoaded = true;

            contentIframe.onload = () => {
                syncFrontendAuth();
                if (loadingIndicator) {
                    loadingIndicator.style.display = 'none';
                }
            };
        } else if (sectionParam) {
            // Dashboard already loaded — postMessage to navigate in-place
            navigateInDashboard(sectionParam);
            // Hide loading indicator immediately for in-place navigation
            if (loadingIndicator) {
                loadingIndicator.style.display = 'none';
            }
        }
    }, 900);
}

function syncFrontendAuth() {
    if (!contentIframe.contentWindow || !authSession.token) return;

    let targetOrigin = '*';
    try {
        targetOrigin = new URL(frontendUrl).origin;
    } catch {
        // Fall back to wildcard if the frontend URL is malformed.
    }

    contentIframe.contentWindow.postMessage({
        type: 'desktop-auth',
        token: authSession.token,
        role: authSession.role,
    }, targetOrigin);
}

// Send a navigation message to the already-loaded content iframe
function navigateInDashboard(section) {
    try {
        contentIframe.contentWindow.postMessage({ type: 'navigate', section }, '*');
    } catch (e) {
        console.warn('navigate postMessage failed:', e);
    }
}

// Tell the dashboard to open a named modal
function openDashboardModal(modal) {
    try {
        contentIframe.contentWindow.postMessage({ type: 'open-modal', modal }, '*');
    } catch (e) {
        console.warn('open-modal postMessage failed:', e);
    }
}

// Desktop → Menu: fade HUD out, reveal menu (no iframe reload — constellation stays)
function backToMenu() {
    menuOverlay.style.display   = 'flex';
    menuOverlay.style.opacity   = '0';
    menuOverlay.style.transition = 'opacity 0.6s ease';
    hudContainer.style.transition = 'opacity 0.6s ease';

    requestAnimationFrame(() => {
        hudContainer.style.opacity = '0';
        menuOverlay.style.opacity  = '1';
    });

    setTimeout(() => {
        hudContainer.style.display = 'none';
    }, 700);
}

// ─── IPC from main process ────────────────────────────────────────────────────

// Called right after the window loads — gives us the backend URL for login
api.onInitServerUrl((url) => {
    serverUrl = url;
});

// Main process confirmed login + menu server is ready
api.onShowMenu(({ menuUrl }) => {
    transitionToMenu(menuUrl);
});

// Menu animation done — reveal the HUD desktop frame
api.onShowDesktop(({ frontendUrl: url }) => {
    transitionToDesktop(url, null);
});

// Frontend connection status updates (used in desktop phase)
electronAPI.onFrontendStatus((status) => {
    const el = document.getElementById('frontend-status');
    if (el) {
        el.textContent = status;
        el.className = 'value ' + (status === 'CONNECTED' ? 'active' : '');
    }
});

// ─── postMessage from menu iframe or content iframe ───────────────────────────

window.addEventListener('message', (event) => {
    const { type, section } = event.data || {};

    if (type === 'navigate') {
        // Icon clicked in the constellation menu — go to that dashboard section
        if (section === 'arena') {
            // Navigate the whole window to the KNIRVARENA app
            window.location.href = '/arena';
        } else if (section === 'p2p-webgui') {
            // Special action: open the P2P Transport WebGUI modal in the dashboard
            if (!desktopLoaded) {
                // Load desktop first (no specific tab), then open modal once loaded
                transitionToDesktop(serverUrl, null);
                setTimeout(() => openDashboardModal('p2p-webgui'), 950);
            } else {
                transitionToDesktop(serverUrl, null);
                setTimeout(() => openDashboardModal('p2p-webgui'), 950);
            }
        } else if (!desktopLoaded) {
            // First visit: encode section in URL for the app to read on mount
            transitionToDesktop(serverUrl, section);
        } else {
            // Dashboard already loaded: show it and navigate in-place
            menuOverlay.style.display   = 'flex';
            menuOverlay.style.opacity   = '1';
            // Trigger back-to-desktop reveal then navigate
            transitionToDesktop(serverUrl, null);
            setTimeout(() => navigateInDashboard(section), 950);
        }

    } else if (type === 'back-to-menu') {
        // Back button in the dashboard clicked
        backToMenu();

    } else if (type === 'menu-complete') {
        // Legacy: kept in case anything still sends this
        electronAPI.send('menu-complete');
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
            authSession = { token, role };
            localStorage.setItem('knirv_nexus_token', token);
            localStorage.setItem('knirv_nexus_role', role);
            localStorage.setItem('knirv_auth_token', token);
            localStorage.setItem('knirv_auth_role', role);
        }

        // Tell main — it starts the menu server and replies with show-menu
        electronAPI.send('login-success');

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

document.getElementById('menu-back-btn').addEventListener('click', () => {
    backToMenu();
});

document.getElementById('minimize-btn').addEventListener('click', () => {
    electronAPI.send('minimize-window');
});

document.getElementById('close-btn').addEventListener('click', () => {
    electronAPI.send('close-window');
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
    const uptime  = electronAPI.getSystemMetrics().uptime;
    const hours   = Math.floor(uptime / 3600);
    const minutes = Math.floor((uptime % 3600) / 60);
    const el      = document.getElementById('uptime');
    if (el) el.textContent = `${hours}h ${minutes}m`;
}

let previousCpuInfo = electronAPI.getSystemMetrics().cpus;
function getCpuUsage() {
    const cpus = electronAPI.getSystemMetrics().cpus;
    if (!cpus.length) return 0;
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
    const { totalMem, freeMem } = electronAPI.getSystemMetrics();
    if (!totalMem) return 0;
    return Math.round(((totalMem - freeMem) / totalMem) * 100);
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

    // Batch canvas drawing with requestAnimationFrame to avoid layout thrashing
    requestAnimationFrame(() => {
        drawPerformanceChart();
    });

    // Simulate network and disk activity based on CPU usage for more realistic metrics
    const baseActivity = cpu / 100;
    const net  = Math.min(100, Math.max(0, 30 + baseActivity * 40 + Math.random() * 10 - 5));
    const disk = Math.min(100, Math.max(0, 40 + baseActivity * 30 + Math.random() * 15 - 7.5));
    setStyle('network-bar', 'width', `${net}%`);
    setEl('network-value', `${Math.round(net)}%`);
    setStyle('disk-bar', 'width', `${disk}%`);
    setEl('disk-value', `${Math.round(disk)}%`);

    // Network throughput correlated with activity
    const netRx = (baseActivity * 800 + Math.random() * 200).toFixed(0);
    const netTx = (baseActivity * 400 + Math.random() * 100).toFixed(0);
    setEl('net-rx', `${netRx} KB/s`);
    setEl('net-tx', `${netTx} KB/s`);

    // Disk I/O correlated with activity
    const diskRead = (baseActivity * 40 + Math.random() * 10).toFixed(1);
    const diskWrite = (baseActivity * 25 + Math.random() * 5).toFixed(1);
    setEl('disk-read', `${diskRead} MB/s`);
    setEl('disk-write', `${diskWrite} MB/s`);

    // Process count with some correlation to system load
    const procs = 180 + Math.floor(baseActivity * 50) + Math.floor(Math.random() * 40);
    setStyle('process-bar', 'width', `${(procs / 400) * 100}%`);
    setEl('process-value', `${procs}`);
    setEl('threads', `${Math.floor(procs * 2.2 + Math.random() * 20)}`);

    // Temperature and fan speed correlated with CPU usage
    const temp = 35 + Math.floor(baseActivity * 25) + Math.floor(Math.random() * 5);
    const fan = 1800 + Math.floor(baseActivity * 1500) + Math.floor(Math.random() * 200);
    setEl('temp', `${temp}°C`);
    setEl('fan', `${fan} RPM`);
}

function initSystemInfo() {
    const osEl   = document.getElementById('os-info');
    const archEl = document.getElementById('arch-info');
    const info = electronAPI.getSystemInfo();
    if (osEl)   osEl.textContent   = `${info.type} ${info.release}`.trim();
    if (archEl) archEl.textContent = info.arch;
}

// Start HUD monitoring once (metrics only matter in the desktop phase,
// but we can run them in the background from the start — low overhead)
initSystemInfo();
updateTime();
updateUptime();
updateMetrics();
setInterval(updateTime,    1000);
setInterval(updateUptime, 60000);
setInterval(updateMetrics, 3000); // Reduced frequency for less critical metrics
drawPerformanceChart();

console.log('KNIRV Desktop renderer initialised — waiting for login.');
