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
const isUpdate = process.argv[2] === 'update';

function runNpmUpdate() {
  const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm';
  console.log('Updating Gmit to the latest npm release...');
  const child = spawn(npmCommand, ['install', '--global', 'gmit@latest'], {
    stdio: 'inherit',
    shell: process.platform === 'win32'
  });
  child.on('error', (error) => {
    console.error(`Gmit update failed: ${error.message}`);
    process.exit(1);
  });
  child.on('exit', (code) => {
    if (code === 0) {
      console.log('Gmit update completed. Run `gmit version` to verify the installed version.');
    }
    process.exit(code ?? 1);
  });
}

function runBinary(args, onExit) {
  const child = spawn(binaryPath, args, { stdio: 'inherit' });
  child.on('error', (error) => {
    console.error(`Unable to start Gmit: ${error.message}`);
    process.exit(1);
  });
  child.on('exit', (code) => onExit(code ?? 1));
}

if (!fs.existsSync(binaryPath) && !isUpdate) {
  console.error(`Binary not found: ${binaryPath}`);
  process.exit(1);
}

if (isUpdate && !fs.existsSync(binaryPath)) {
  runNpmUpdate();
} else if (isUpdate) {
  // Let the current binary show its version/update notice, then exit it before
  // npm replaces the package files (required on Windows where the .exe is locked).
  runBinary(['version'], () => runNpmUpdate());
} else {
  runBinary(process.argv.slice(2), (code) => process.exit(code));
}
