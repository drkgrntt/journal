// The shell loads the live site in a webview; nothing from the web app is
// bundled, so a normal server deploy updates the app for everyone.
//
// CONTOUR_URL comes from the environment or from a file in mobile/. Day to day
// that's .env (copy .env.example; gitignored, your dev server). For a release
// build run with CONTOUR_ENV=production (npm run sync:prod) to read the committed
// .env.production instead. A variable already set in your shell wins over either.
//
// The user agent suffix below is how the server recognises the shell
// (middleware.DetectNativeApp): it hides marketing links and billing controls.
const fs = require('fs');
const path = require('path');
const envFile = path.join(__dirname, process.env.CONTOUR_ENV === 'production' ? '.env.production' : '.env');
if (fs.existsSync(envFile) && typeof process.loadEnvFile === 'function') {
  process.loadEnvFile(envFile); // Node 20.12+; never overrides variables already set
}

const url = process.env.CONTOUR_URL;
if (!url) {
  throw new Error('Set CONTOUR_URL (in mobile/.env, see .env.example) to the address the app should load');
}

/** @type {import('@capacitor/cli').CapacitorConfig} */
module.exports = {
  appId: 'app.contourjournal',
  appName: 'Contour Journal',
  webDir: 'www',
  appendUserAgent: 'ContourApp/1.0',
  server: {
    url,
    // Plain http is for the emulator only; a release URL must be https.
    cleartext: url.startsWith('http://'),
    // Keep the webview on the app's own origin. Anything else (Stripe, mail
    // links) opens outside it.
    allowNavigation: [new URL(url).hostname],
  },
  plugins: {
    SplashScreen: { launchAutoHide: true, backgroundColor: '#fafafa' },
    StatusBar: { style: 'DEFAULT' },
  },
};
