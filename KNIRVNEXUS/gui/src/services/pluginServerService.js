// Plugin Server Service for KNIRV-NEXUS
// Handles communication with the plugin server for WASM and binary file management

const PLUGIN_SERVER_BASE_URL = 'http://localhost:8082';

class PluginServerService {
  constructor() {
    this.baseUrl = PLUGIN_SERVER_BASE_URL;
  }

  // Get server information
  async getServerInfo() {
    try {
      const response = await fetch(`${this.baseUrl}/info`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('Failed to get server info:', error);
      throw error;
    }
  }

  // List all available agents
  async listAgents() {
    try {
      const response = await fetch(`${this.baseUrl}/list`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('Failed to list agents:', error);
      throw error;
    }
  }

  // Upload a new agent file
  async uploadAgent(file, onProgress = null) {
    try {
      const formData = new FormData();
      formData.append('plugin-agent', file);

      const xhr = new XMLHttpRequest();
      
      return new Promise((resolve, reject) => {
        xhr.upload.addEventListener('progress', (event) => {
          if (event.lengthComputable && onProgress) {
            const percentComplete = (event.loaded / event.total) * 100;
            onProgress(percentComplete);
          }
        });

        xhr.addEventListener('load', () => {
          if (xhr.status === 200) {
            try {
              const response = JSON.parse(xhr.responseText);
              resolve(response);
            } catch (e) {
              reject(new Error('Invalid JSON response'));
            }
          } else {
            reject(new Error(`Upload failed with status: ${xhr.status}`));
          }
        });

        xhr.addEventListener('error', () => {
          reject(new Error('Upload failed'));
        });

        xhr.open('POST', `${this.baseUrl}/upload`);
        xhr.send(formData);
      });
    } catch (error) {
      console.error('Failed to upload agent:', error);
      throw error;
    }
  }

  // Download an agent file
  async downloadAgent(agentName) {
    try {
      const response = await fetch(`${this.baseUrl}/agents/${agentName}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      // Return the blob for download
      const blob = await response.blob();
      return {
        blob,
        filename: agentName,
        size: blob.size
      };
    } catch (error) {
      console.error('Failed to download agent:', error);
      throw error;
    }
  }

  // Delete an agent file
  async deleteAgent(agentName) {
    try {
      const response = await fetch(`${this.baseUrl}/delete/${agentName}`, {
        method: 'DELETE'
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('Failed to delete agent:', error);
      throw error;
    }
  }

  // Check if plugin server is available
  async isServerAvailable() {
    try {
      const response = await fetch(`${this.baseUrl}/info`, {
        method: 'GET',
        signal: AbortSignal.timeout(5000) // 5 second timeout
      });
      return response.ok;
    } catch (error) {
      console.warn('Plugin server not available:', error.message);
      return false;
    }
  }

  // Get agent file URL for direct access
  getAgentUrl(agentName) {
    return `${this.baseUrl}/agents/${agentName}`;
  }

  // Validate file type for agent uploads
  isValidAgentFile(file) {
    const validExtensions = ['.wasm', '.so', '.dll', '.dylib'];
    const fileName = file.name.toLowerCase();
    return validExtensions.some(ext => fileName.endsWith(ext));
  }

  // Get file type description
  getFileTypeDescription(fileName) {
    const ext = fileName.toLowerCase().split('.').pop();
    switch (ext) {
      case 'wasm':
        return 'WebAssembly Module';
      case 'so':
        return 'Linux Shared Library';
      case 'dll':
        return 'Windows Dynamic Library';
      case 'dylib':
        return 'macOS Dynamic Library';
      default:
        return 'Unknown File Type';
    }
  }

  // Format file size for display
  formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
}

// Export singleton instance
export const pluginServerService = new PluginServerService();
export default pluginServerService;
