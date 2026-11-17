// On-device drafts for new journal entries, so writing survives a dropped
// connection. autosave.js writes the form's state here on every input, then
// tries to save it to the server; a draft is deleted only once the server has
// confirmed the exact text it holds. Anything still here (offline, server
// error, tab closed mid-save) is retried by flushDrafts() - on page load, on
// reconnect, and when the tab comes back.
//
// Drafts are plaintext in localStorage, unlike entries at rest on the server.
// So they are short-lived by design: removed the moment they sync, and wiped
// on logout (see offline.js).
//
// New entries only: an existing entry is saved straight to the server, since
// replaying an offline edit over a copy changed elsewhere would silently
// overwrite it.

import { currentUserId } from "./index.js"

const PREFIX = "journal-draft:"

// Drafts being actively saved by a live autosave, mapped to the form that
// owns them. flushDrafts() leaves those to autosave so the two never send the
// same draft at once - until the form leaves the page (an htmx swap), after
// which nothing is handling it and it's fair game.
const activeDrafts = new Map()

export function markDraftActive(id, form) {
	activeDrafts.set(id, form)
}

function isActive(id) {
	const form = activeDrafts.get(id)
	if (form?.isConnected) return true
	activeDrafts.delete(id)
	return false
}

export function newDraftId() {
	if (crypto.randomUUID) return crypto.randomUUID()
	return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

function storageKey(userId, id) {
	return `${PREFIX}${userId}:${id}`
}

// localStorage can throw (private windows, blocked site data, quota). Drafts
// are a safety net on top of the normal server save, so failing quietly just
// means that net isn't there.
function safely(fn, fallback) {
	try {
		return fn()
	} catch {
		return fallback
	}
}

export function saveDraft(draft) {
	return safely(() => {
		localStorage.setItem(storageKey(draft.userId, draft.id), JSON.stringify({ ...draft, updatedAt: Date.now() }))
		return true
	}, false)
}

export function loadDraft(userId, id) {
	return safely(() => JSON.parse(localStorage.getItem(storageKey(userId, id))), null)
}

export function removeDraft(userId, id) {
	safely(() => localStorage.removeItem(storageKey(userId, id)))
}

export function listDrafts(userId) {
	return safely(() => {
		const drafts = []
		const prefix = `${PREFIX}${userId}:`
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i)
			if (!key || !key.startsWith(prefix)) continue
			const draft = JSON.parse(localStorage.getItem(key))
			if (draft) drafts.push(draft)
		}
		return drafts
	}, [])
}

// Every user's drafts, for logout.
export function removeAllDrafts() {
	safely(() => {
		const keys = []
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i)
			if (key?.startsWith(PREFIX)) keys.push(key)
		}
		keys.forEach((key) => localStorage.removeItem(key))
	})
}

// Sends one entry save. Resolves to:
//   { status: "saved", entry }   the server has it
//   { status: "retry" }          offline or a server error - try again later
//   { status: "auth" }           session expired - keep the draft, retry after login
//   { status: "rejected" }       the server refused it (4xx) - retrying won't help
export async function sendEntry(serverId, body) {
	let res
	try {
		// "manual": an expired session is answered with a redirect to the
		// login page, which must not be mistaken for a successful save.
		res = await fetch(serverId ? `/api/journal/${serverId}` : "/api/journal", {
			method: serverId ? "PUT" : "POST",
			// body is a string, which fetch would send as text/plain - the
			// server only parses form bodies (a URLSearchParams object would
			// set this itself).
			headers: { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" },
			body,
			credentials: "same-origin",
			redirect: "manual",
		})
	} catch {
		return { status: "retry" }
	}

	if (res.type === "opaqueredirect" || res.status === 401) return { status: "auth" }
	if (res.status >= 500) return { status: "retry" }
	if (!res.ok) return { status: "rejected" }

	try {
		return { status: "saved", entry: await res.json() }
	} catch {
		return { status: "retry" }
	}
}

let flushing = false
// A draft the server rejected is retried once per page load (not on every
// flush), so a bad draft can't loop but one rejected for a since-fixed
// reason still gets another chance.
const attempted = new Set()

// Sends every draft that no live autosave is handling. Returns how many are
// still waiting afterward.
export async function flushDrafts() {
	const userId = currentUserId()
	if (!userId || flushing || navigator.onLine === false) return pendingDrafts().length
	flushing = true

	// One tab at a time, so two tabs can't both POST the same draft.
	const run = async () => {
		for (const draft of listDrafts(userId)) {
			if (isActive(draft.id) || (draft.rejected && attempted.has(draft.id))) continue
			attempted.add(draft.id)

			const result = await sendEntry(draft.serverId, draft.body)
			// Offline, a server error, or a lapsed session: stop, and the
			// next reconnect/load/login tries again.
			if (result.status === "auth" || result.status === "retry") break

			if (result.status === "rejected") {
				saveDraft({ ...draft, rejected: true })
				continue
			}

			// Typing may have updated this draft while it was in flight - only
			// delete it if it's still exactly what the server just confirmed.
			const latest = loadDraft(userId, draft.id)
			if (!latest || latest.body === draft.body) {
				removeDraft(userId, draft.id)
			} else {
				saveDraft({ ...latest, serverId: result.entry.id })
			}
		}
	}

	try {
		if (navigator.locks) {
			await navigator.locks.request("journal-draft-flush", run)
		} else {
			await run()
		}
	} finally {
		flushing = false
	}
	return pendingDrafts().length
}

// Every draft the server doesn't have yet, including ones it rejected.
export function pendingDrafts() {
	const userId = currentUserId()
	return userId ? listDrafts(userId) : []
}

function flushWhenPossible() {
	if (document.visibilityState === "visible") flushDrafts()
}

window.addEventListener("online", flushWhenPossible)
document.addEventListener("visibilitychange", flushWhenPossible)
// A draft left behind by a closed tab or an earlier page load.
document.addEventListener("DOMContentLoaded", flushWhenPossible)
