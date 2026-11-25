// On-device outbox for action items and thankfuls, so they can be added,
// completed and deleted without a connection. Each change is queued here
// (localStorage, per user) and replayed to the server in order by
// flushOutbox() - on page load, on reconnect, and when the tab comes back.
// The queueing rules (coalescing, id mapping) live in outboxQueue.js.
//
// Ops hold plaintext text on the device, unlike items at rest on the server.
// So they are short-lived by design: removed the moment they sync, and wiped
// on logout (see offline.js), like journal drafts.
//
// Events on window: "outbox:change" after any change to the queue, and
// "outbox:synced" (detail { op, serverId }) after the server accepts an op.

import { currentUserId } from "./index.js"
import { emptyRecord, enqueue, buildRequest, applyResult, pendingCount, nextOps, markSent, classifyResponse, dropRejected } from "./outboxQueue.js"

const PREFIX = "outbox:"
const SEND_TIMEOUT_MS = 15000
// A rejected op holds plaintext, so it is not kept forever.
const REJECTED_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000

function storageKey(userId) {
	return `${PREFIX}${userId}`
}

function newId() {
	if (crypto.randomUUID) return crypto.randomUUID()
	return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

// localStorage can throw (private windows, blocked site data, quota). The
// outbox is a convenience on top of the normal online save, so failing quietly
// just means offline changes aren't kept.
function safely(fn, fallback) {
	try {
		return fn()
	} catch {
		return fallback
	}
}

function loadRecord(userId) {
	return safely(() => {
		const raw = JSON.parse(localStorage.getItem(storageKey(userId)))
		if (!raw || !Array.isArray(raw.ops)) return emptyRecord()
		return { ops: raw.ops, idMap: raw.idMap && typeof raw.idMap === "object" ? raw.idMap : {} }
	}, emptyRecord())
}

function saveRecord(userId, record) {
	safely(() => {
		// An empty queue with nothing left to resolve leaves no key behind.
		if (record.ops.length === 0 && Object.keys(record.idMap).length === 0) {
			localStorage.removeItem(storageKey(userId))
		} else {
			localStorage.setItem(storageKey(userId), JSON.stringify(record))
		}
	})
}

function notifyChange() {
	window.dispatchEvent(new CustomEvent("outbox:change"))
}

export function onlineNow() {
	return navigator.onLine !== false
}

// Queues a change. `partial` is { kind, type, ref, text?, completedAt? };
// for a create, ref.clientId is generated here. Returns the op, or null when
// nobody is signed in.
export function enqueueOp(partial) {
	const userId = currentUserId()
	if (!userId) return null

	const op = { ...partial, id: newId(), createdAt: Date.now() }
	op.ref = { ...partial.ref }
	if (op.type === "create" && !op.ref.clientId) op.ref.clientId = newId()

	saveRecord(userId, enqueue(loadRecord(userId), op))
	notifyChange()
	if (onlineNow()) flushOutbox()
	return op
}

// Sends one op. Resolves to { status, serverId? }:
//   saved     the server has it
//   gone      a PUT/DELETE target no longer exists - nothing left to do
//   retry     offline, timed out, a server error, or a create answered without
//             an id - try again later
//   auth      session expired - keep the op, retry after login
//   rejected  the server refused it (other 4xx) - retrying won't help
async function sendOp(op, request) {
	const controller = new AbortController()
	// A hung request would hold the flush lock (and every tab's flush) forever.
	const timer = setTimeout(() => controller.abort(), SEND_TIMEOUT_MS)
	try {
		// "manual": an expired session is answered with a redirect to the
		// login page, which must not be mistaken for a successful save.
		const res = await fetch(request.url, {
			method: request.method,
			// A string body would go out as text/plain - the server only
			// parses form bodies.
			headers: { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" },
			body: request.method === "DELETE" ? undefined : request.body,
			credentials: "same-origin",
			redirect: "manual",
			signal: controller.signal,
		})
		return classifyResponse(op, res, res.headers.get("X-Item-Id"))
	} catch {
		return { status: "retry" }
	} finally {
		clearTimeout(timer)
	}
}

let flushing = false
// A rejected op is retried once per page load (not on every flush), so a bad
// one can't loop but one rejected for a since-fixed reason gets another go.
const attempted = new Set()

// Sends every eligible op in order. Returns how many are still waiting.
export async function flushOutbox() {
	const userId = currentUserId()
	if (!userId || flushing || !onlineNow()) return pendingOps().length
	flushing = true

	// One tab at a time, so two tabs can't both send the same op.
	const run = async () => {
		// Ops skipped or sent in this pass, so each is looked at once.
		const seen = new Set()
		saveRecord(userId, dropRejected(loadRecord(userId), REJECTED_MAX_AGE_MS, Date.now()))
		for (;;) {
			// Re-read every time: ops enqueued mid-flush must not be lost.
			const record = loadRecord(userId)
			const op = nextOps(record, attempted).find((o) => !seen.has(o.id))
			if (!op) break
			seen.add(op.id)

			const request = buildRequest(op, record.idMap)
			// Its item hasn't synced yet; a later flush will have the id.
			if (!request) continue
			attempted.add(op.id)
			// Persist "sent" before the request: if the response is lost, the
			// create must stay queued (not be cancelled by a later delete)
			// because the server may already have the row.
			if (op.type === "create" && !op.sent) saveRecord(userId, markSent(loadRecord(userId), op.id))

			const result = await sendOp(op, request)
			// Offline, a server error, or a lapsed session: stop, and the
			// next reconnect/load/login tries again.
			if (result.status === "auth" || result.status === "retry") break

			saveRecord(userId, applyResult(loadRecord(userId), op, result))
			notifyChange()
			if (result.status === "saved") {
				window.dispatchEvent(new CustomEvent("outbox:synced", { detail: { op, serverId: result.serverId } }))
			}
		}
	}

	try {
		if (navigator.locks) {
			await navigator.locks.request("outbox-flush", run)
		} else {
			await run()
		}
	} finally {
		flushing = false
	}
	return pendingOps().length
}

// Every change the server doesn't have yet, including ones it rejected.
export function pendingOps() {
	const userId = currentUserId()
	return userId ? loadRecord(userId).ops : []
}

export function pendingOutboxCount() {
	const userId = currentUserId()
	return userId ? pendingCount(loadRecord(userId)) : 0
}

// Every user's outbox, for logout.
export function removeAllOutbox() {
	safely(() => {
		const keys = []
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i)
			if (key?.startsWith(PREFIX)) keys.push(key)
		}
		keys.forEach((key) => localStorage.removeItem(key))
	})
}

function flushWhenPossible() {
	if (document.visibilityState === "visible") flushOutbox()
}

window.addEventListener("online", flushWhenPossible)
document.addEventListener("visibilitychange", flushWhenPossible)
// Changes left behind by a closed tab or an earlier page load.
document.addEventListener("DOMContentLoaded", flushWhenPossible)
