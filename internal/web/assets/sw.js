// Contour Journal service worker. Served from /sw.js (serviceWorker.controller.go),
// which stamps the build's asset version in below.
//
// Its one job is making "write now, sync later" possible: it keeps the
// new-entry page and the static assets it needs available with no connection,
// so the form (and autosave's on-device drafts, see assets/js/utils/drafts.js)
// opens anywhere. It deliberately caches nothing else - in particular no
// dashboard, journal list or entry pages, since those hold decrypted entries.

const VERSION = "__ASSET_VERSION__"
const ASSETS_CACHE = `contour-assets-${VERSION}`
const PAGES_CACHE = `contour-pages-${VERSION}`
const CACHE_PREFIX = "contour-"

const WRITE_URL = "/journal/new"
// Opening the installed app lands on /dashboard (the manifest's start_url),
// whose Today tab is this same form - offline, that opens the new-entry page.
const SHELL_PATHS = new Set([WRITE_URL, "/dashboard"])

// How long a navigation waits on the network before falling back to the
// cached page. Offline on a phone often means a request that hangs rather
// than one that fails fast.
const NAVIGATION_TIMEOUT_MS = 4000

self.addEventListener("install", () => self.skipWaiting())

// Caches from older versions are deleted by the "warm" message, not here:
// until the current version's caches are filled, the old ones are the only
// offline copy there is.
self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()))

self.addEventListener("message", (event) => {
	if (event.data?.type === "warm") {
		event.waitUntil(warm(event.data.urls || []))
	}
})

async function warm(urls) {
	try {
		const pages = await caches.open(PAGES_CACHE)
		const res = await fetch(WRITE_URL, { credentials: "same-origin" })
		// A redirect means the session is gone (login page) - never cache that.
		if (res.ok && !res.redirected) await pages.put(WRITE_URL, res)

		const assets = await caches.open(ASSETS_CACHE)
		await Promise.all(
			urls.map(async (url) => {
				const parsed = new URL(url, self.location.origin)
				if (parsed.origin !== self.location.origin || !parsed.pathname.startsWith("/assets/")) return
				if (await assets.match(parsed.href)) return
				try {
					await assets.add(parsed.href)
				} catch {
					// One missing asset shouldn't stop the rest.
				}
			}),
		)

		const keys = await caches.keys()
		await Promise.all(
			keys
				.filter((key) => key.startsWith(CACHE_PREFIX) && key !== ASSETS_CACHE && key !== PAGES_CACHE)
				.map((key) => caches.delete(key)),
		)
	} catch {
		// Offline or logged out: warming is best effort and retried on a later load.
	}
}

self.addEventListener("fetch", (event) => {
	const request = event.request
	if (request.method !== "GET") return

	const url = new URL(request.url)
	if (url.origin !== self.location.origin) return

	if (url.pathname.startsWith("/assets/")) {
		event.respondWith(assetResponse(request, url))
	} else if (request.mode === "navigate" && SHELL_PATHS.has(url.pathname)) {
		event.respondWith(shellResponse(request, url))
	}
})

// Versioned URLs (?v=<build hash>) never change content, so cache-first is
// safe. Anything unversioned (icons, the manifest) goes network-first so an
// updated file isn't pinned behind a stale copy.
async function assetResponse(request, url) {
	const cache = await caches.open(ASSETS_CACHE)

	if (url.searchParams.has("v")) {
		const cached = await cache.match(request)
		if (cached) return cached
		const res = await fetch(request)
		if (res.ok) cache.put(request, res.clone())
		return res
	}

	try {
		const res = await fetch(request)
		if (res.ok) cache.put(request, res.clone())
		return res
	} catch (err) {
		const cached = await cache.match(request)
		if (cached) return cached
		throw err
	}
}

async function shellResponse(request, url) {
	const pages = await caches.open(PAGES_CACHE)
	const cached = await pages.match(WRITE_URL)

	const network = fetch(request).then((res) => {
		if (url.pathname === WRITE_URL && res.ok && !res.redirected) pages.put(WRITE_URL, res.clone())
		return res
	})

	if (!cached) return network

	// Keep the in-flight request alive past the fallback so a slow response
	// still refreshes the cache.
	network.catch(() => {})
	const timeout = new Promise((_, reject) => setTimeout(() => reject(new Error("timeout")), NAVIGATION_TIMEOUT_MS))
	try {
		return await Promise.race([network, timeout])
	} catch {
		return cached
	}
}
