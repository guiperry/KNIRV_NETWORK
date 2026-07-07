#!/usr/bin/env node
import { cpSync, copyFileSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const rootDir = resolve(scriptDir, '..');
const buildDir = join(rootDir, 'build');

const androidApk = join(buildDir, 'knirvcontroller-android-latest.apk');
const iosUnsignedIpa = join(buildDir, 'knirvconroller-ios-unsigned.ipa');
const iosIpa = iosUnsignedIpa;
const iosSigningScript = join(rootDir, 'scripts', 'sign-ios-app.sh');
const buildSigningScript = join(buildDir, 'sign-ios-app.sh');
const androidSigningDir = join(buildDir, 'signing');
const androidKeystore = join(androidSigningDir, 'android-release.keystore');

const args = new Set(process.argv.slice(2));
const androidOnly = args.has('--android-only');
const iosOnly = args.has('--ios-only');
const uploadOnly = args.has('--upload-only');
const noUpload = args.has('--no-upload');
const skipSync = args.has('--skip-sync');

if (androidOnly && iosOnly) {
  throw new Error('Choose only one of --android-only or --ios-only.');
}

const buildAndroid = !uploadOnly && !iosOnly;
const buildIos = !uploadOnly && !androidOnly;
const uploadExistingAndroid = uploadOnly ? !iosOnly : buildAndroid;
const uploadExistingIos = uploadOnly ? !androidOnly : buildIos;

function run(command, commandArgs, cwd = rootDir) {
  console.log(`> ${command} ${commandArgs.join(' ')}`);
  execFileSync(command, commandArgs, {
    cwd,
    stdio: 'inherit',
    env: process.env,
  });
}

function runWithEnv(command, commandArgs, cwd, envOverrides) {
  console.log(`> ${command} ${commandArgs.join(' ')}`);
  execFileSync(command, commandArgs, {
    cwd,
    stdio: 'inherit',
    env: {
      ...process.env,
      ...envOverrides,
    },
  });
}

function ensureDir(path) {
  mkdirSync(path, { recursive: true });
}

function cleanFile(path) {
  if (existsSync(path)) {
    rmSync(path, { force: true });
  }
}

function copyIosSigningScript() {
  if (!existsSync(iosSigningScript)) {
    return;
  }

  ensureDir(buildDir);
  copyFileSync(iosSigningScript, buildSigningScript);
}

function uploadArtifact(artifactPath, remotePath) {
  if (!existsSync(artifactPath)) {
    throw new Error(`Missing release artifact: ${artifactPath}`);
  }

  run('rclone', ['copy', artifactPath, remotePath]);
}

function buildAndroidRelease() {
  console.log('Building Android release APK...');
  const signingEnv = prepareAndroidSigningEnv();
  runWithEnv('./gradlew', ['assembleRelease'], join(rootDir, 'android'), signingEnv);

  const releaseDir = join(rootDir, 'android', 'app', 'build', 'outputs', 'apk', 'release');
  const signedApk = join(releaseDir, 'app-release.apk');
  const unsignedApk = join(releaseDir, 'app-release-unsigned.apk');
  const sourceApk = existsSync(signedApk) ? signedApk : unsignedApk;

  if (!existsSync(sourceApk)) {
    throw new Error(`Android release build did not produce an APK in ${releaseDir}.`);
  }

  ensureDir(buildDir);
  copyFileSync(sourceApk, androidApk);
  console.log(`Android package ready: ${androidApk}`);
}

function prepareAndroidSigningEnv() {
  const customSigning = process.env.KNIRV_ANDROID_KEYSTORE_FILE
    && process.env.KNIRV_ANDROID_KEYSTORE_PASSWORD
    && process.env.KNIRV_ANDROID_KEY_ALIAS
    && process.env.KNIRV_ANDROID_KEY_PASSWORD;

  if (customSigning) {
    return {};
  }

  ensureDir(androidSigningDir);

  if (!existsSync(androidKeystore)) {
    run('keytool', [
      '-genkeypair',
      '-v',
      '-keystore',
      androidKeystore,
      '-alias',
      'androiddebugkey',
      '-storepass',
      'android',
      '-keypass',
      'android',
      '-dname',
      'CN=Android Debug,O=Android,C=US',
      '-keyalg',
      'RSA',
      '-keysize',
      '2048',
      '-validity',
      '10000',
    ], androidSigningDir);
  }

  return {
    KNIRV_ANDROID_KEYSTORE_FILE: androidKeystore,
    KNIRV_ANDROID_KEYSTORE_PASSWORD: 'android',
    KNIRV_ANDROID_KEY_ALIAS: 'androiddebugkey',
    KNIRV_ANDROID_KEY_PASSWORD: 'android',
  };
}

function buildIosRelease() {
  if (process.platform !== 'darwin') {
    throw new Error('iOS IPA builds require macOS and Xcode.');
  }

  console.log('Building unsigned iOS release IPA...');

  const iosBuildDir = join(buildDir, 'ios-release');
  const derivedDataPath = join(iosBuildDir, 'DerivedData');
  const productsDir = join(derivedDataPath, 'Build', 'Products', 'Release-iphoneos');
  const appBundle = join(productsDir, 'App.app');
  const payloadDir = join(iosBuildDir, 'Payload');
  const stagedAppBundle = join(payloadDir, 'App.app');

  rmSync(iosBuildDir, { recursive: true, force: true });
  ensureDir(iosBuildDir);

  run('xcodebuild', [
    '-project',
    'ios/App/App.xcodeproj',
    '-scheme',
    'App',
    '-configuration',
    'Release',
    '-destination',
    'generic/platform=iOS',
    '-derivedDataPath',
    derivedDataPath,
    'CODE_SIGNING_ALLOWED=NO',
    'CODE_SIGNING_REQUIRED=NO',
    'CODE_SIGN_IDENTITY=',
    'build',
  ]);

  if (!existsSync(appBundle)) {
    throw new Error(`iOS release build did not produce an app bundle at ${appBundle}.`);
  }

  ensureDir(buildDir);
  ensureDir(payloadDir);
  cpSync(appBundle, stagedAppBundle, { recursive: true });
  cleanFile(iosUnsignedIpa);

  run('zip', ['-qry', iosUnsignedIpa, 'Payload'], iosBuildDir);
  copyIosSigningScript();
  console.log(`Unsigned iOS package ready: ${iosUnsignedIpa}`);
}

function main() {
  ensureDir(buildDir);
  copyIosSigningScript();
  if (buildIos && process.platform !== 'darwin') {
    throw new Error('iOS IPA builds require macOS and Xcode. Use --android-only on non-mac hosts.');
  }

  if (!uploadOnly && !skipSync && (buildAndroid || buildIos)) {
    run('npm', ['run', 'cap:sync']);
  }

  if (buildAndroid) {
    cleanFile(androidApk);
  }

  if (buildIos) {
    cleanFile(iosIpa);
  }

  if (buildAndroid) {
    buildAndroidRelease();
  }

  if (buildIos) {
    buildIosRelease();
  }

  if (!noUpload && uploadExistingAndroid) {
    uploadArtifact(androidApk, 'incline:knirv/controller/android/');
  }

  if (!noUpload && uploadExistingIos) {
    uploadArtifact(iosIpa, 'incline:knirv/controller/ios/');
  }

  if (uploadOnly) {
    console.log(noUpload ? 'Existing release artifacts checked.' : 'Existing release artifacts uploaded.');
    return;
  }

  console.log(noUpload ? 'Release artifacts packaged.' : 'Release artifacts uploaded.');
}

main();
