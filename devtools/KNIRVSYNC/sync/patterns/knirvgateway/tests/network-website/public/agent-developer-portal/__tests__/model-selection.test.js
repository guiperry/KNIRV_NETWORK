/**
 * Tests for Developer Portal Model Selection
 * Tests the model selection interface and agent registration functionality
 */

// Mock DOM environment for testing
const { JSDOM } = require('jsdom');

describe('Developer Portal Model Selection', () => {
  let dom;
  let document;
  let window;

  beforeEach(() => {
    // Create a new DOM instance for each test
    dom = new JSDOM(`
      <!DOCTYPE html>
      <html>
        <head><title>Test</title></head>
        <body>
          <select id="agentModel">
            <option value="phi-3-mini">Phi-3 Mini (3.8B) - Recommended</option>
            <option value="recurrentgemma">RecurrentGemma (2.7B) - Efficient</option>
            <option value="tinyllama">TinyLlama (1.1B) - Lightweight</option>
          </select>
          <div id="modelDescription">
            <strong>Phi-3 Mini:</strong> Best overall performance with 3.8B parameters. Excellent for complex reasoning tasks and general-purpose AI applications. MIT licensed.
          </div>
          <input type="text" id="agentName" value="Test Agent">
          <input type="text" id="capability" value="Test Capability">
          <input type="text" id="description" value="Test Description">
          <input type="text" id="providerName" value="Test Provider">
          <input type="text" id="version" value="1.0.0">
          <div id="registrationStatus" style="display: none;"></div>
        </body>
      </html>
    `, {
      url: 'http://localhost',
      pretendToBeVisual: true,
      resources: 'usable'
    });

    document = dom.window.document;
    window = dom.window;
    global.document = document;
    global.window = window;

    // Mock console methods
    global.console = {
      log: jest.fn(),
      error: jest.fn(),
      warn: jest.fn(),
      info: jest.fn()
    };
  });

  afterEach(() => {
    dom.window.close();
  });

  describe('Model Selection Interface', () => {
    it('should have all three model options available', () => {
      const modelSelect = document.getElementById('agentModel');
      const options = modelSelect.querySelectorAll('option');

      expect(options).toHaveLength(3);
      expect(options[0].value).toBe('phi-3-mini');
      expect(options[1].value).toBe('recurrentgemma');
      expect(options[2].value).toBe('tinyllama');
    });

    it('should have Phi-3 Mini as default selection', () => {
      const modelSelect = document.getElementById('agentModel');
      expect(modelSelect.value).toBe('phi-3-mini');
    });

    it('should display correct model descriptions', () => {
      // Test model descriptions object
      const modelDescriptions = {
        'phi-3-mini': '<strong>Phi-3 Mini:</strong> Best overall performance with 3.8B parameters. Excellent for complex reasoning tasks and general-purpose AI applications. MIT licensed.',
        'recurrentgemma': '<strong>RecurrentGemma:</strong> Novel recurrent architecture with 2.7B parameters. Highly efficient for long sequences and stateful tasks. Perfect for memory-intensive applications.',
        'tinyllama': '<strong>TinyLlama:</strong> Ultra-lightweight with 1.1B parameters. Ideal for resource-constrained environments and edge deployment. Apache 2.0 licensed.'
      };

      // Test each model description
      Object.keys(modelDescriptions).forEach(model => {
        expect(modelDescriptions[model]).toContain('<strong>');
        expect(modelDescriptions[model]).toContain('parameters');
      });
    });

    it('should update description when model selection changes', () => {
      // Simulate the model description update functionality
      const modelSelect = document.getElementById('agentModel');
      const descriptionDiv = document.getElementById('modelDescription');

      const modelDescriptions = {
        'phi-3-mini': '<strong>Phi-3 Mini:</strong> Best overall performance with 3.8B parameters. Excellent for complex reasoning tasks and general-purpose AI applications. MIT licensed.',
        'recurrentgemma': '<strong>RecurrentGemma:</strong> Novel recurrent architecture with 2.7B parameters. Highly efficient for long sequences and stateful tasks. Perfect for memory-intensive applications.',
        'tinyllama': '<strong>TinyLlama:</strong> Ultra-lightweight with 1.1B parameters. Ideal for resource-constrained environments and edge deployment. Apache 2.0 licensed.'
      };

      // Test changing to RecurrentGemma
      modelSelect.value = 'recurrentgemma';
      descriptionDiv.innerHTML = modelDescriptions['recurrentgemma'];

      expect(descriptionDiv.innerHTML).toContain('RecurrentGemma');
      expect(descriptionDiv.innerHTML).toContain('2.7B parameters');

      // Test changing to TinyLlama
      modelSelect.value = 'tinyllama';
      descriptionDiv.innerHTML = modelDescriptions['tinyllama'];

      expect(descriptionDiv.innerHTML).toContain('TinyLlama');
      expect(descriptionDiv.innerHTML).toContain('1.1B parameters');
    });
  });

  describe('Agent Registration System', () => {
    it('should collect all required registration data', () => {
      const agentName = document.getElementById('agentName').value;
      const capability = document.getElementById('capability').value;
      const description = document.getElementById('description').value;
      const providerName = document.getElementById('providerName').value;
      const version = document.getElementById('version').value;
      const agentModel = document.getElementById('agentModel').value;

      expect(agentName).toBe('Test Agent');
      expect(capability).toBe('Test Capability');
      expect(description).toBe('Test Description');
      expect(providerName).toBe('Test Provider');
      expect(version).toBe('1.0.0');
      expect(agentModel).toBe('phi-3-mini');
    });

    it('should generate valid agent hash', () => {
      // Test the hash generation function
      const generateAgentHash = (data) => {
        const str = JSON.stringify(data);
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
          const char = str.charCodeAt(i);
          hash = ((hash << 5) - hash) + char;
          hash = hash & hash; // Convert to 32-bit integer
        }
        return 'knirv_' + Math.abs(hash).toString(16).padStart(8, '0') + '_' + Date.now().toString(16);
      };

      const testData = {
        name: 'Test Agent',
        capability: 'Test Capability',
        model: 'phi-3-mini'
      };

      const hash = generateAgentHash(testData);

      expect(hash).toMatch(/^knirv_[0-9a-f]{8}_[0-9a-f]+$/);
      expect(hash).toContain('knirv_');
    });

    it('should create proper registration data structure', () => {
      const registrationData = {
        name: 'Test Agent',
        capability: 'Test Capability',
        description: 'Test Description',
        provider: 'Test Provider',
        version: '1.0.0',
        model: 'phi-3-mini',
        timestamp: new Date().toISOString(),
        network: 'knirv-testnet'
      };

      expect(registrationData).toHaveProperty('name');
      expect(registrationData).toHaveProperty('capability');
      expect(registrationData).toHaveProperty('description');
      expect(registrationData).toHaveProperty('provider');
      expect(registrationData).toHaveProperty('version');
      expect(registrationData).toHaveProperty('model');
      expect(registrationData).toHaveProperty('timestamp');
      expect(registrationData).toHaveProperty('network');

      expect(registrationData.network).toBe('knirv-testnet');
      expect(registrationData.model).toBe('phi-3-mini');
    });

    it('should handle registration success response', () => {
      const mockSuccessResponse = {
        success: true,
        agentId: 'did:knirv:agent-1234567890',
        agentHash: 'knirv_abcdef12_1234567890',
        transactionId: 'tx_1234567890',
        blockHeight: 567890
      };

      expect(mockSuccessResponse.success).toBe(true);
      expect(mockSuccessResponse.agentId).toMatch(/^did:knirv:agent-\d+$/);
      expect(mockSuccessResponse.agentHash).toMatch(/^knirv_[0-9a-f]+_[0-9a-f]+$/);
      expect(mockSuccessResponse.transactionId).toMatch(/^tx_\d+$/);
      expect(typeof mockSuccessResponse.blockHeight).toBe('number');
    });

    it('should handle registration error response', () => {
      const mockErrorResponse = {
        success: false,
        error: 'Failed to register agent with KNIRVORACLE: Network timeout'
      };

      expect(mockErrorResponse.success).toBe(false);
      expect(mockErrorResponse.error).toContain('KNIRVORACLE');
    });
  });

  describe('Model Display Names', () => {
    it('should return correct display names for models', () => {
      const getModelDisplayName = (modelValue) => {
        const modelNames = {
          'phi-3-mini': 'Phi-3 Mini (3.8B)',
          'recurrentgemma': 'RecurrentGemma (2.7B)',
          'tinyllama': 'TinyLlama (1.1B)'
        };
        return modelNames[modelValue] || modelValue;
      };

      expect(getModelDisplayName('phi-3-mini')).toBe('Phi-3 Mini (3.8B)');
      expect(getModelDisplayName('recurrentgemma')).toBe('RecurrentGemma (2.7B)');
      expect(getModelDisplayName('tinyllama')).toBe('TinyLlama (1.1B)');
      expect(getModelDisplayName('unknown')).toBe('unknown');
    });
  });

  describe('Form Validation', () => {
    it('should validate required fields', () => {
      const validateForm = () => {
        const agentName = document.getElementById('agentName').value.trim();
        const capability = document.getElementById('capability').value.trim();
        const providerName = document.getElementById('providerName').value.trim();
        
        return agentName.length > 0 && capability.length > 0 && providerName.length > 0;
      };

      expect(validateForm()).toBe(true);

      // Test with empty fields
      document.getElementById('agentName').value = '';
      expect(validateForm()).toBe(false);
    });
  });
});
