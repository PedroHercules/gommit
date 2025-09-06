#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

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

const { osName, archName, ext } = getPlatform();
const binaryName = `gmit-${osName}-${archName}${ext}`;
const binaryPath = path.join(__dirname, 'bin', binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error(`Binary not found: ${binaryPath}`);
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

child.on('exit', (code) => {
  process.exit(code);
});