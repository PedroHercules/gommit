const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const GITHUB_REPO = 'PedroHercules/gommit';
const VERSION = require('./package.json').version;

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;
  
  let osName, archName, ext;
  
  switch (platform) {
    case 'win32':
      osName = 'windows';
      ext = '.exe';
      break;
    case 'darwin':
      osName = 'darwin';
      ext = '';
      break;
    case 'linux':
      osName = 'linux';
      ext = '';
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }
  
  switch (arch) {
    case 'x64':
      archName = 'amd64';
      break;
    case 'arm64':
      archName = 'arm64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }
  
  return { osName, archName, ext };
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Handle redirect
        return downloadFile(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
      
      file.on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    }).on('error', (err) => {
      reject(err);
    });
  });
}

async function install() {
  try {
    console.log('Installing gommit...');
    
    const { osName, archName, ext } = getPlatform();
    const binaryName = `gommit-${osName}-${archName}${ext}`;
    const downloadUrl = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${binaryName}`;
    
    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    const binaryPath = path.join(binDir, `gommit${ext}`);
    
    console.log(`Downloading ${downloadUrl}...`);
    await downloadFile(downloadUrl, binaryPath);
    
    // Make executable on Unix systems
    if (process.platform !== 'win32') {
      fs.chmodSync(binaryPath, '755');
    }
    
    console.log('gommit installed successfully!');
    console.log('\nUsage:');
    console.log('  gommit commit          # Generate commit message');
    console.log('  gommit config set-key  # Set OpenRouter API key');
    console.log('  gommit --help          # Show help');
    
  } catch (error) {
    console.error('Installation failed:', error.message);
    console.error('\nPlease try:');
    console.error('1. Check your internet connection');
    console.error('2. Verify the release exists on GitHub');
    console.error('3. Install manually from: https://github.com/' + GITHUB_REPO + '/releases');
    process.exit(1);
  }
}

if (require.main === module) {
  install();
}

module.exports = { install };