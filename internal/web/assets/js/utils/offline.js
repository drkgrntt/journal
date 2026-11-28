// Registers the service worker (assets/sw.js, served from /sw.js), keeps its
// offline copy of the new-entry page fresh, and cleans up on-device data at
// logout. See drafts.js for the drafts themselves.

import { currentUserId } from "./index.js"
import { flushDrafts, pendingDrafts, removeAllDrafts } from "./drafts.js"
import { flushOutbox, pendingOps, removeAllOutbox } from "./outbox.js"

const WARM_INTERVAL_MS = 6 * 60 * 60 * 1000
const WARMED_KEY = "offline-warmed-at"

async function warmOfflineCopy(registration) {
	if (!currentUserId() || navigator.onLine === false) return

	try {
		const last = Number(localStorage.getItem(`${WARMED_KEY}-${currentUserId()}`) || 0)
		if (Date.now() - last < WARM_INTERVAL_MS) return
		localStorage.setItem(`${WARMED_KEY}-${currentUserId()}`, String(Date.now()))
	} catch {
		// Without storage this just warms on every load, which is harmless.
	}

	// The page has already loaded everything it needs - hand the worker that
	// list instead of making it guess at the dependency graph. These come out
	// of the browser's own cache, not the network.
	const urls = performance
		.getEntriesByType("resource")
		.map((entry) => entry.name)
		.filter((name) => new URL(name).pathname.startsWith("/assets/"))

	registration.active?.postMessage({ type: "warm", urls })
}

async function registerServiceWorker() {
	if (!("serviceWorker" in navigator)) return
	try {
		await navigator.serviceWorker.register("/sw.js")
		const registration = await navigator.serviceWorker.ready
		warmOfflineCopy(registration)
	} catch {
		// No worker means no offline page - everything else still works.
	}
}

window.addEventListener("load", registerServiceWorker)

async function clearOfflineData() {
	removeAllDrafts()
	removeAllOutbox()
	try {
		const keys = await caches.keys()
		await Promise.all(keys.filter((key) => key.startsWith("contour-")).map((key) => caches.delete(key)))
		Object.keys(localStorage)
			.filter((key) => key.startsWith(WARMED_KEY) || key.startsWith("appearance-") || key.startsWith("grace-banner-dismissed-"))
			.forEach((key) => localStorage.removeItem(key))
	} catch {
		// Nothing to clear.
	}
}

const SIGN_OUT_EVERYWHERE_PATH = "/api/profile/sign-out-everywhere"
const SIGN_OUT_PATHS = ["/api/auth/logout", SIGN_OUT_EVERYWHERE_PATH]

// Logging out (or signing out everywhere) wipes the on-device drafts and queued changes, and the cached page (which contains
// the user's outstanding action items). Drafts that haven't synced get one
// last attempt first, and a warning if they still can't - logging out would
// otherwise silently throw them away.
document.addEventListener("htmx:confirm", async (event) => {
	if (!SIGN_OUT_PATHS.includes(event.detail.path)) return

	event.preventDefault()
	// No hx-confirm on the form: confirm.js would answer it too and send twice.
	if (event.detail.path === SIGN_OUT_EVERYWHERE_PATH) {
		const sure = await window.confirmDialog(
			"Sign out of every browser and device, including this one?",
			"Log out everywhere",
		)
		if (!sure) return
	}
	if (pendingDrafts().length > 0 || pendingOps().length > 0) {
		const remaining = (await flushDrafts()) + (await flushOutbox())
		if (remaining > 0) {
			const noun = remaining === 1 ? "change hasn't" : "changes haven't"
			const proceed = await window.confirmDialog(
				`${remaining} ${noun} synced yet. Logging out will discard ${remaining === 1 ? "it" : "them"}.`,
				"Log out anyway",
			)
			if (!proceed) return
		}
	}

	await clearOfflineData()
	event.detail.issueRequest(true)
})

// A deleted account leaves nothing behind on this device either. This waits for
// the server to confirm, so a mistyped password doesn't cost any drafts.
document.addEventListener("htmx:afterRequest", async (event) => {
	if (event.detail.pathInfo?.requestPath !== "/api/profile/delete-account" || !event.detail.successful) return
	await clearOfflineData()
	window.location.assign("/")
})
