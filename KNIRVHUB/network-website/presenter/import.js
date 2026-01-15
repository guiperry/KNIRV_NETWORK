class PresentationImporter {
    constructor() {
        this.initializeElements();
        this.bindEvents();
        this.currentImportData = null;
    }

    initializeElements() {
        this.fileUpload = document.getElementById('fileUpload');
        this.fileInput = document.getElementById('fileInput');
        this.importForm = document.getElementById('importForm');
        this.configForm = document.getElementById('configForm');
        this.progressSection = document.getElementById('progressSection');
        this.progressFill = document.getElementById('progressFill');
        this.progressText = document.getElementById('progressText');
        this.logOutput = document.getElementById('logOutput');
        this.cancelBtn = document.getElementById('cancelBtn');

        // Form inputs
        this.presentationName = document.getElementById('presentationName');
        this.presentationPassword = document.getElementById('presentationPassword');
        this.presentationDescription = document.getElementById('presentationDescription');
        this.folderName = document.getElementById('folderName');
    }

    bindEvents() {
        this.fileUpload.addEventListener('click', () => this.fileInput.click());
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
        
        // Drag and drop
        this.fileUpload.addEventListener('dragover', (e) => {
            e.preventDefault();
            this.fileUpload.classList.add('dragover');
        });
        
        this.fileUpload.addEventListener('dragleave', () => {
            this.fileUpload.classList.remove('dragover');
        });
        
        this.fileUpload.addEventListener('drop', (e) => {
            e.preventDefault();
            this.fileUpload.classList.remove('dragover');
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleFileSelect({ target: { files } });
            }
        });

        this.configForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.submitImport();
        });

        this.cancelBtn.addEventListener('click', () => this.hideImportForm());
    }



    async handleFileSelect(event) {
        const file = event.target.files[0];
        if (!file || !file.name.endsWith('.zip')) {
            alert('Please select a valid ZIP file.');
            return;
        }

        this.currentImportData = { file, type: 'zip' };

        // Extract folder name from zip file name
        const folderName = file.name.replace('.zip', '').replace(/[^a-zA-Z0-9]/g, '');

        this.presentationName.value = folderName.replace(/([A-Z])/g, ' $1').trim();
        this.folderName.value = folderName;
        this.presentationPassword.value = this.generatePassword();
        this.presentationDescription.value = `Imported presentation from ${file.name}`;

        this.showImportForm();
    }

    showImportForm() {
        this.importForm.classList.remove('hidden');
        this.importForm.scrollIntoView({ behavior: 'smooth' });
    }

    hideImportForm() {
        this.importForm.classList.add('hidden');
        this.currentImportData = null;
    }

    async submitImport() {
        if (!this.currentImportData) return;

        this.showProgress();
        this.updateProgress(20, 'Preparing import...');

        try {
            // Check if we're running with the standalone presenter server or through KNIRVORACLE
            const isStandalonePresenter = await this.checkIfStandalonePresenter();

            if (isStandalonePresenter) {
                // Standalone presenter server - use the full import functionality
                this.addLog('🔧 Standalone presenter server detected');
                this.addLog('📡 Using server-side import processing');
                await this.performLocalImport();
            } else {
                // KNIRVORACLE or deployed version - try Netlify functions first
                this.addLog('☁️ KNIRVORACLE/Netlify environment detected');
                this.addLog('🔄 Attempting import via Netlify functions...');
                await this.performNetlifyImport();
            }

        } catch (error) {
            console.error('Import error:', error);
            this.updateProgress(0, 'Import failed: ' + error.message);
            this.addLog('❌ Import failed: ' + error.message);
        }
    }

    async performLocalImport() {
        this.updateProgress(30, 'Uploading file...');

        const formData = new FormData();
        formData.append('presentationName', this.presentationName.value);
        formData.append('password', this.presentationPassword.value);
        formData.append('description', this.presentationDescription.value);
        formData.append('folderName', this.folderName.value);
        formData.append('type', this.currentImportData.type);

        if (this.currentImportData.type === 'zip') {
            formData.append('zipFile', this.currentImportData.file);
        }

        this.updateProgress(50, 'Processing presentation...');

        const response = await fetch('/api/import-presentation', {
            method: 'POST',
            body: formData
        });

        const result = await response.json();

        if (result.success) {
            this.updateProgress(80, 'Applying responsive design...');
            this.addLog('✅ Presentation imported successfully');
            this.addLog(`📁 Folder: ${result.folder}`);
            this.addLog(`📊 Slides: ${result.slides}`);

            this.updateProgress(100, 'Import completed successfully!');
            this.addLog(`🔗 Share URL: ${window.location.origin}${this.getBasePath()}share.html?p=${result.folder}`);
            this.addLog('✨ New presentation card will appear with dropdown menu');

            setTimeout(() => {
                this.addLog('🔄 Redirecting to main page...');
                setTimeout(() => {
                    const basePath = this.getBasePath();
                    window.location.href = `${basePath}index.html`;
                }, 1000);
            }, 2000);
        } else {
            throw new Error(result.error || 'Import failed');
        }
    }

    async checkIfStandalonePresenter() {
        try {
            // Try to access a presenter-specific endpoint that only exists in standalone mode
            const response = await fetch('/api/slide-exists/test/1', { method: 'HEAD' });
            // If this succeeds, we're running the standalone presenter server
            return response.status !== 404;
        } catch (error) {
            // If this fails, we're probably in KNIRVORACLE mode
            return false;
        }
    }

    async performNetlifyImport() {
        this.updateProgress(30, 'Connecting to Netlify functions...');

        const formData = new FormData();
        formData.append('presentationName', this.presentationName.value);
        formData.append('password', this.presentationPassword.value);
        formData.append('description', this.presentationDescription.value);
        formData.append('folderName', this.folderName.value);
        formData.append('type', this.currentImportData.type);

        if (this.currentImportData.type === 'zip') {
            formData.append('zipFile', this.currentImportData.file);
        }

        this.addLog('📤 Sending request to Netlify function...');
        this.updateProgress(50, 'Processing via Netlify...');

        const response = await fetch('/api/import-presentation', {
            method: 'POST',
            body: formData
        });

        const result = await response.json();

        if (result.success) {
            this.updateProgress(100, 'Import completed successfully!');
            this.addLog('✅ Presentation imported successfully');
            this.addLog(`📁 Folder: ${result.folder}`);
            this.addLog(`📊 Slides: ${result.slides}`);
            this.addLog(`🔗 Share URL: ${window.location.origin}${this.getBasePath()}share.html?p=${result.folder}`);
            this.addLog('✨ New presentation card will appear with dropdown menu');

            setTimeout(() => {
                this.addLog('🔄 Redirecting to main page...');
                setTimeout(() => {
                    const basePath = this.getBasePath();
                    window.location.href = `${basePath}index.html`;
                }, 1000);
            }, 2000);
        } else {
            // Netlify function returned an error (likely indicating manual setup needed)
            this.updateProgress(100, 'Manual setup required');
            this.addLog('📋 ' + result.error);
            this.addLog('💡 ' + result.message);

            setTimeout(() => {
                this.addLog('🔄 Redirecting to manual setup guide...');
                setTimeout(() => {
                    const basePath = this.getBasePath();
                    window.location.href = result.redirectUrl || `${basePath}manual-setup.html`;
                }, 2000);
            }, 2000);
        }
    }

    showProgress() {
        this.importForm.classList.add('hidden');
        this.progressSection.classList.remove('hidden');
        this.progressSection.scrollIntoView({ behavior: 'smooth' });
        this.updateProgress(10, 'Starting import...');
    }

    updateProgress(percent, text) {
        this.progressFill.style.width = percent + '%';
        this.progressText.textContent = text;
    }

    addLog(message) {
        const timestamp = new Date().toLocaleTimeString();
        this.logOutput.innerHTML += `[${timestamp}] ${message}\n`;
        this.logOutput.scrollTop = this.logOutput.scrollHeight;
    }

    generatePassword() {
        const adjectives = ['secure', 'private', 'protected', 'safe', 'hidden'];
        const nouns = ['presentation', 'slides', 'deck', 'content'];
        const numbers = Math.floor(Math.random() * 1000);
        
        const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
        const noun = nouns[Math.floor(Math.random() * nouns.length)];
        
        return `${adj}${noun}${numbers}`;
    }

    getBasePath() {
        // Detect if we're running in KNIRVORACLE context
        const currentPath = window.location.pathname;
        if (currentPath.includes('/presenter')) {
            // We're in KNIRVORACLE, maintain the presenter path
            return '/presenter/';
        } else {
            // We're in standalone presenter mode
            return '';
        }
    }
}

// Initialize the importer
const importer = new PresentationImporter();
