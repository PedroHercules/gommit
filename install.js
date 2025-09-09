const fs = require('fs');
const path = require('path');

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



async function install() {
  try {
    console.log('Installing gmit...');
    
    const { osName, archName, ext } = getPlatform();
    const sourceBinaryName = `gmit-${osName}-${archName}${ext}`;
    
    const binDir = path.join(__dirname, 'bin');
    const sourceBinaryPath = path.join(binDir, sourceBinaryName);
    
    // Check if the platform-specific binary exists
    if (!fs.existsSync(sourceBinaryPath)) {
      throw new Error(`Binary not found for platform ${osName}-${archName}. Available binaries: ${fs.readdirSync(binDir).join(', ')}`);
    }
    
    // Make executable on Unix systems
    console.log(`Setting up binary for ${osName}-${archName}...`);
    if (process.platform !== 'win32') {
      fs.chmodSync(sourceBinaryPath, 0o755);
    }
    
    console.log('gmit installed successfully!');
    console.log('\nUsage:');
    console.log('  gmit commit          # Generate commit message');
    console.log('  gmit config set-key  # Set OpenRouter API key');
    console.log('  gmit --help          # Show help');
    
  } catch (error) {
    console.error('Installation failed:', error.message);
    console.error('\nPlease try:');
    console.error('1. Check if the binary exists for your platform');
    console.error('2. Verify your system architecture is supported');
    console.error('3. Install manually from: https://github.com/PedroHercules/gommit/releases');
    process.exit(1);
  }
}

if (require.main === module) {
  install();
}

module.exports = { install };