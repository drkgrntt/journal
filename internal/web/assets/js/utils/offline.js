// Registers the service worker (assets/sw.js, served from /sw.js), keeps its
// offline copy of the new-entry page fresh, and cleans up on-device data at
// logout. See drafts.js for the drafts themselves.

import { currentUserId } from "./index.js"
import { flushDrafts, pendingDrafts, removeAllDrafts } from "./drafts.js"

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
	try {
		const keys = await caches.keys()
		await Promise.all(keys.filter((key) => key.startsWith("contour-")).map((key) => caches.delete(key)))
		Object.keys(localStorage)
			.filter((key) => key.startsWith(WARMED_KEY))
			.forEach((key) => localStorage.removeItem(key))
	} catch {
		// Nothing to clear.
	}
}

// Logging out wipes the on-device drafts and the cached page (which contains
// the user's outstanding action items). Drafts that haven't synced get one
// last attempt first, and a warning if they still can't - logging out would
// otherwise silently throw them away.
document.addEventListener("htmx:confirm", async (event) => {
	if (event.detail.path !== "/api/auth/logout") return

	event.preventDefault()
	if (pendingDrafts().length > 0) {
		const remaining = await flushDrafts()
		if (remaining > 0) {
			const noun = remaining === 1 ? "entry hasn't" : "entries haven't"
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
