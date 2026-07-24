// KNIRV Developer Portal — Developer Key Generation
// Wallet-signed developer.key / verifyer.key issuance, backed by the
// onboarding.knirv.com session + registration API (CORS is open on that API).
(() => {
  const ONBOARDING_ORIGIN = 'https://onboarding.knirv.com';
  const API_SESSION = `${ONBOARDING_ORIGIN}/api/onboarding`;
  const API_REGISTER_KEY = `${ONBOARDING_ORIGIN}/api/register-key`;

  const state = {
    keyType: 'developer',
    sessionId: '',
    walletAddress: '',
    keyUrl: '',
    keyFileName: ''
  };

  const els = {};

  function planForKeyType(keyType) {
    return keyType === 'verifyer' ? 'enterprise' : 'pro';
  }

  function signMessageTemplate(wallet, keyType, sessionId) {
    return `KNIRV Network Key Registration\n\nPlan: ${planForKeyType(keyType)}\nKey Type: ${keyType}\nWallet: ${wallet}\nSession: ${sessionId}\n\nBy signing this message you authorize KNIRV to issue a ${keyType}.key bound to this wallet address.`;
  }

  function setStatus(message, tone = 'neutral') {
    if (!els.status) return;
    els.status.textContent = message;
    els.status.style.color = tone === 'error' ? '#ff8a8a' : tone === 'ok' ? '#86efac' : 'var(--transparent-white-7)';
  }

  async function ensureSession() {
    if (state.sessionId) return state.sessionId;
    const res = await fetch(API_SESSION, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        plan: planForKeyType(state.keyType),
        source: 'developer-portal',
        stage: 'dev-key-init',
        keyType: state.keyType
      })
    });
    if (!res.ok) throw new Error(`Could not start a key session (HTTP ${res.status})`);
    const data = await res.json();
    state.sessionId = data.sessionId;
    return state.sessionId;
  }

  async function connectWallet() {
    if (!window.ethereum) {
      setStatus('No Ethereum wallet detected. Install MetaMask or another EIP-1193 wallet.', 'error');
      return;
    }
    setStatus('Connecting wallet…');
    try {
      await ensureSession();
      const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
      if (!accounts || !accounts.length) throw new Error('No accounts returned from wallet.');
      state.walletAddress = accounts[0];
      setStatus(`Wallet connected: ${state.walletAddress}`, 'ok');
      els.connectBtn.style.display = 'none';
      els.signBtn.style.display = '';
    } catch (error) {
      setStatus(error instanceof Error ? error.message : 'Wallet connection failed.', 'error');
    }
  }

  async function signAndGenerate() {
    setStatus('Waiting for signature…');
    try {
      const message = signMessageTemplate(state.walletAddress, state.keyType, state.sessionId);
      const signature = await window.ethereum.request({ method: 'personal_sign', params: [message, state.walletAddress] });
      setStatus('Generating key file…');
      const res = await fetch(API_REGISTER_KEY, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sessionId: state.sessionId,
          plan: planForKeyType(state.keyType),
          keyType: state.keyType,
          walletAddress: state.walletAddress,
          signature,
          nonce: state.sessionId,
          message
        })
      });
      if (!res.ok) {
        const failure = await res.json().catch(() => ({}));
        throw new Error(failure.error || `Key generation failed (HTTP ${res.status})`);
      }
      const blob = await res.blob();
      if (state.keyUrl) URL.revokeObjectURL(state.keyUrl);
      state.keyUrl = URL.createObjectURL(blob);
      state.keyFileName = `${state.keyType}.key`;
      setStatus('Key file ready — click Download key.', 'ok');
      els.signBtn.style.display = 'none';
      els.downloadBtn.style.display = '';
    } catch (error) {
      setStatus(error instanceof Error ? error.message : 'Signing failed.', 'error');
    }
  }

  function downloadKey() {
    if (!state.keyUrl) return;
    const a = document.createElement('a');
    a.href = state.keyUrl;
    a.download = state.keyFileName || `${state.keyType}.key`;
    a.click();
  }

  function resetFlow() {
    state.sessionId = '';
    state.walletAddress = '';
    if (state.keyUrl) URL.revokeObjectURL(state.keyUrl);
    state.keyUrl = '';
    state.keyFileName = '';
    els.connectBtn.style.display = '';
    els.signBtn.style.display = 'none';
    els.downloadBtn.style.display = 'none';
  }

  function selectKeyType(keyType, buttons) {
    if (keyType === state.keyType) return;
    state.keyType = keyType;
    resetFlow();
    buttons.forEach((btn) => {
      const active = btn.getAttribute('data-key-type') === keyType;
      btn.classList.toggle('active', active);
      btn.style.borderColor = active ? 'var(--bright-blue)' : 'var(--transparent-white-2)';
      btn.style.backgroundColor = active ? 'rgba(0, 192, 250, 0.08)' : 'var(--transparent-white-05)';
    });
    setStatus(keyType === 'verifyer'
      ? 'Verifyer Key requires an Enterprise account.'
      : 'Developer Key requires a Pro or Enterprise account.');
  }

  document.addEventListener('DOMContentLoaded', () => {
    els.status = document.getElementById('dev-key-status');
    els.connectBtn = document.getElementById('dev-key-connect-btn');
    els.signBtn = document.getElementById('dev-key-sign-btn');
    els.downloadBtn = document.getElementById('dev-key-download-btn');
    if (!els.connectBtn) return; // section not present on this page

    const typeButtons = Array.from(document.querySelectorAll('[data-key-type]'));
    typeButtons.forEach((btn) => {
      btn.addEventListener('click', () => selectKeyType(btn.getAttribute('data-key-type'), typeButtons));
    });

    els.connectBtn.addEventListener('click', connectWallet);
    els.signBtn.addEventListener('click', signAndGenerate);
    els.downloadBtn.addEventListener('click', downloadKey);
  });
})();
