# Contour Journal: native shell

A thin [Capacitor](https://capacitorjs.com) wrapper. It bundles none of the
web app; it opens the live site in a webview, so deploying the server updates
the app. The server recognises the shell by the `ContourApp/` user agent suffix
(`middleware.DetectNativeApp`) and, for it only:

- redirects `/` to `/dashboard` (no marketing landing page),
- hides the About and Blog footer links,
- hides the plan purchase and manage buttons on the profile page.

The user agent is client-controlled, so this is presentation, not enforcement.

## First run (Android)

```bash
cd mobile
npm install
cp .env.example .env          # then set CONTOUR_URL to your dev server's port
npx cap add android
npx cap sync
npx cap open android          # run from Android Studio
```

`CONTOUR_URL` (read from `mobile/.env`, or from your shell, which wins) is the
address the webview loads; `10.0.2.2` is your machine as seen from the emulator.
Re-run `npx cap sync` whenever it changes. Use an `https://` URL for anything you ship.

iOS is the same with `npx cap add ios` (needs macOS and Xcode).

## Release builds

`npm run sync:prod` points the app at `https://contourjournal.com` (from the
committed `.env.production`) instead of your dev server. Run `npm run sync:dev`
(or plain `npx cap sync`) to go back. The synced URL is baked into the native
project, so re-sync before every build you intend to ship or test.

## Check on a device

- Sign in, reload, and confirm the session survives (the `x-token` cookie).
- Write offline, then reconnect, and confirm the draft syncs. Also check
  whether the service worker registers on iOS; if not, drafts still work from
  `localStorage` but the offline `/journal/new` page isn't cached.
- Check safe-area insets and the keyboard on the new-entry page.

## Before store submission

- Icons and splash: generate with `@capacitor/assets` from the `assets/images` icons.
- Billing: Apple and Google restrict selling digital plans inside apps. Hiding
  the buttons is the web-only-checkout route; check the current store rules.
- Privacy policy (`privacy.templ`) should mention the app.
- Consider a native touch (biometric lock) for Apple guideline 4.2.
