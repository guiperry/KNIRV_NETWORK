/**
 * Tests for WASM Deployment Documentation
 * Validates the WASM deployment sequence documentation and functionality
 */

const fs = require('fs');
const path = require('path');

describe('WASM Deployment Documentation', () => {
  let deploymentPageContent;

  beforeAll(() => {
    // Read the deployment page content
    const deploymentPagePath = path.join(__dirname, '../getting-started-step4.html');
    deploymentPageContent = fs.readFileSync(deploymentPagePath, 'utf8');
  });

  describe('Documentation Content', () => {
    it('should contain WASM deployment section', () => {
      expect(deploymentPageContent).toContain('WASM Agent Core Build Deployment Sequence');
      expect(deploymentPageContent).toContain('WebAssembly');
      expect(deploymentPageContent).toContain('D-TEN network');
    });

    it('should document all four build pipeline steps', () => {
      expect(deploymentPageContent).toContain('Agent Core Compilation');
      expect(deploymentPageContent).toContain('WASM Optimization & Validation');
      expect(deploymentPageContent).toContain('Deployment Package Creation');
      expect(deploymentPageContent).toContain('Network Deployment & Registration');
    });

    it('should include technical requirements section', () => {
      expect(deploymentPageContent).toContain('Technical Requirements');
      expect(deploymentPageContent).toContain('WASM Runtime');
      expect(deploymentPageContent).toContain('Security');
      expect(deploymentPageContent).toContain('Network Integration');
    });

    it('should document WASI requirements', () => {
      expect(deploymentPageContent).toContain('WebAssembly System Interface (WASI)');
      expect(deploymentPageContent).toContain('Memory limit: 512MB - 2GB');
      expect(deploymentPageContent).toContain('Execution timeout: 30s per invocation');
    });

    it('should document security requirements', () => {
      expect(deploymentPageContent).toContain('Sandboxed execution environment');
      expect(deploymentPageContent).toContain('No direct file system access');
      expect(deploymentPageContent).toContain('Network access via KNIRV APIs only');
    });

    it('should document network integration requirements', () => {
      expect(deploymentPageContent).toContain('KNIRVORACLE registration required');
      expect(deploymentPageContent).toContain('LoRA adapter compatibility');
      expect(deploymentPageContent).toContain('Real-time skill invocation support');
    });

    it('should include command reference section', () => {
      expect(deploymentPageContent).toContain('Command Reference');
      expect(deploymentPageContent).toContain('knirv build --target=wasm');
      expect(deploymentPageContent).toContain('knirv deploy --env=testnet');
      expect(deploymentPageContent).toContain('knirv status --deployment-id');
    });
  });

  describe('Build Pipeline Steps', () => {
    it('should document step 1: Agent Core Compilation', () => {
      expect(deploymentPageContent).toContain('TypeScript agent-core is compiled to WebAssembly');
      expect(deploymentPageContent).toContain('KNIRVCONTROLLER build pipeline');
      expect(deploymentPageContent).toContain('knirv build --target=wasm --model=');
      expect(deploymentPageContent).toContain('Compiling cognitive shell templates');
      expect(deploymentPageContent).toContain('Embedding model weights');
      expect(deploymentPageContent).toContain('Generating agent.wasm');
    });

    it('should document step 2: WASM Optimization', () => {
      expect(deploymentPageContent).toContain('Size optimization using wasm-opt');
      expect(deploymentPageContent).toContain('Security validation and sandboxing checks');
      expect(deploymentPageContent).toContain('D-TEN network compatibility verification');
      expect(deploymentPageContent).toContain('LoRA adapter integration validation');
    });

    it('should document step 3: Package Creation', () => {
      expect(deploymentPageContent).toContain('agent-deployment.tar.gz');
      expect(deploymentPageContent).toContain('agent.wasm');
      expect(deploymentPageContent).toContain('manifest.json');
      expect(deploymentPageContent).toContain('config/');
      expect(deploymentPageContent).toContain('metadata.json');
    });

    it('should document step 4: Network Deployment', () => {
      expect(deploymentPageContent).toContain('Upload to secure KNIRVNEXUS environment');
      expect(deploymentPageContent).toContain('Agent registration with KNIRVORACLE');
      expect(deploymentPageContent).toContain('Network capability announcement');
      expect(deploymentPageContent).toContain('Health check and monitoring setup');
    });
  });

  describe('Command Examples', () => {
    it('should provide correct build command syntax', () => {
      expect(deploymentPageContent).toContain('knirv build --target=wasm --model=phi-3-mini --optimize');
    });

    it('should provide correct deployment command syntax', () => {
      expect(deploymentPageContent).toContain('knirv deploy --env=testnet --region=us-east');
    });

    it('should provide correct status monitoring command', () => {
      expect(deploymentPageContent).toContain('knirv status --deployment-id=[deployment-id]');
    });
  });

  describe('Visual Elements', () => {
    it('should include step numbers and icons', () => {
      // Check for step numbering
      expect(deploymentPageContent).toMatch(/<span[^>]*>1<\/span>/);
      expect(deploymentPageContent).toMatch(/<span[^>]*>2<\/span>/);
      expect(deploymentPageContent).toMatch(/<span[^>]*>3<\/span>/);
      expect(deploymentPageContent).toMatch(/<span[^>]*>4<\/span>/);

      // Check for Font Awesome icons
      expect(deploymentPageContent).toContain('fas fa-cogs');
      expect(deploymentPageContent).toContain('fas fa-microchip');
      expect(deploymentPageContent).toContain('fas fa-shield-alt');
      expect(deploymentPageContent).toContain('fas fa-network-wired');
    });

    it('should have proper styling and layout', () => {
      expect(deploymentPageContent).toContain('glass-card');
      expect(deploymentPageContent).toContain('background-color: var(--transparent-white-05)');
      expect(deploymentPageContent).toContain('border-left: 4px solid');
    });
  });

  describe('Content Accuracy', () => {
    it('should reference correct KNIRV components', () => {
      expect(deploymentPageContent).toContain('KNIRVCONTROLLER');
      expect(deploymentPageContent).toContain('KNIRVNEXUS');
      expect(deploymentPageContent).toContain('KNIRVORACLE');
      expect(deploymentPageContent).toContain('D-TEN');
    });

    it('should mention correct file formats and technologies', () => {
      expect(deploymentPageContent).toContain('WebAssembly');
      expect(deploymentPageContent).toContain('WASM');
      expect(deploymentPageContent).toContain('TypeScript');
      expect(deploymentPageContent).toContain('LoRA');
      expect(deploymentPageContent).toContain('tar.gz');
    });

    it('should include realistic memory and timeout limits', () => {
      expect(deploymentPageContent).toContain('512MB - 2GB');
      expect(deploymentPageContent).toContain('30s per invocation');
    });
  });

  describe('Deployment Package Structure', () => {
    it('should document correct package structure', () => {
      const packageStructure = [
        'agent-deployment.tar.gz',
        'agent.wasm',
        'manifest.json',
        'config/',
        'metadata.json'
      ];

      packageStructure.forEach(item => {
        expect(deploymentPageContent).toContain(item);
      });
    });
  });

  describe('Integration Points', () => {
    it('should document integration with other KNIRV components', () => {
      expect(deploymentPageContent).toContain('KNIRVNEXUS DVE');
      expect(deploymentPageContent).toContain('KNIRVORACLE registration');
      expect(deploymentPageContent).toContain('KNIRVCONTROLLER build pipeline');
    });

    it('should mention model integration', () => {
      expect(deploymentPageContent).toContain('model weights');
      expect(deploymentPageContent).toContain('LoRA adapter');
      expect(deploymentPageContent).toContain('[selected-model]');
    });
  });
});
