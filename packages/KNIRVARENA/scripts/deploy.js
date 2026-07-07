import { spawn } from 'child_process';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import fs from 'fs/promises';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const rootDir = join(__dirname, '..');

class Deployer {
  constructor() {
    this.backendUrl = process.env.PROD_API_URL || 'https://knirvcontroller.onrender.com'; // Example Render URL
  }

  async run(command, args, options = {}) {
    return new Promise((resolve, reject) => {
      console.log(`\n> ${command} ${args.join(' ')}`);
      const process = spawn(command, args, {
        stdio: 'inherit',
        shell: true,
        ...options,
      });

      process.on('close', (code) => {
        if (code === 0) {
          resolve(code);
        } else {
          reject(new Error(`Command failed with code ${code}`));
        }
      });

      process.on('error', reject);
    });
  }

  async deployBackend() {
    console.log('\n🚀 Deploying backend server...');
    // This is a placeholder. Replace with your actual deployment command.
    // For example, for Render, this might be a 'git push' if connected to your Git repo.
    // For other services, you might use their CLI.
    console.log('Assuming backend is deployed via Git push to a connected service like Render.');
    console.log(`Backend will be available at: ${this.backendUrl}`);
    // await this.run('your-deployment-command', ['args']);
    console.log('✅ Backend deployment initiated.');
  }

  async createProdEnvFile() {
    console.log(`\n📝 Creating production .env file for PWA build...`);
    const envContent = `VITE_API_BASE_URL=${this.backendUrl}\n`;
    await fs.writeFile(join(rootDir, '.env.production'), envContent);
    console.log(`Created .env.production with VITE_API_BASE_URL=${this.backendUrl}`);
  }

  async buildPWA(platform) {
    if (!['android', 'ios'].includes(platform)) {
      throw new Error(`Invalid platform: ${platform}. Choose 'android' or 'ios'.`);
    }
    console.log(`\n📦 Building PWA for ${platform}...`);

    // The build:pwa script should use the .env.production file
    await this.run('npm', ['run', `build:pwa:${platform}`], { cwd: rootDir });

    console.log(`✅ PWA for ${platform} built successfully.`);
  }

  async cleanup() {
    console.log('\n🧹 Cleaning up...');
    try {
      await fs.unlink(join(rootDir, '.env.production'));
      console.log('Removed .env.production');
    } catch (error) {
      if (error.code !== 'ENOENT') {
        console.warn('Could not remove .env.production:', error.message);
      }
    }
  }

  async execute() {
    const platform = process.argv[2];
    if (!platform) {
      console.error('❌ Please provide a platform: "android" or "ios"');
      console.error('Usage: node scripts/deploy.js <platform>');
      process.exit(1);
    }

    try {
      // 1. Deploy the backend (or ensure it's deployed)
      await this.deployBackend();

      // 2. Create a .env.production file with the live backend URL
      await this.createProdEnvFile();

      // 3. Build the PWA for the specified platform
      await this.buildPWA(platform);

      console.log('\n🎉 Deployment and PWA build process completed successfully!');
      console.log(`You can now deploy the native app from the 'android' or 'ios' directories.`);

    } catch (error) {
      console.error('\n❌ An error occurred during the deployment process:', error.message);
      process.exit(1);
    } finally {
      // 4. Clean up the temporary .env file
      await this.cleanup();
    }
  }
}

const deployer = new Deployer();
deployer.execute();