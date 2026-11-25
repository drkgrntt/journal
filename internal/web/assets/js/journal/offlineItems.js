// Lets action items and thankfuls keep working without a connection.
//
// Online, nothing here runs: htmx sends the request and the server renders the
// result as always. When a request to one of these endpoints can't reach the
// server (htmx:sendError / htmx:timeout), the change goes into the on-device
// outbox (utils/outbox.js) and this file updates the page to match, so the
// list looks the way the person expects. The outbox replays it later.
//
// The open action items on screen come from the service worker's cached copy
// of the new-entry page (see sw.js), which is where "complete it offline"
// finds them. Changes still waiting in the outbox are re-applied to that
// copy on every load, since it can't know about them.
//
// Not covered offline: editing an item (its form is fetched from the server),
// and attaching an item to an entry that doesn't exist on the server yet.
// Items added offline are standalone.

import { enqueueOp, pendingOps } from "../utils/outbox.js"

const UUID = "[0-9a-fA-F-]{36}"
const ROUTES = [
	{ verb: "post", re: /^\/api\/action-items\/?$/, kind: "action", type: "create" },
	{ verb: "put", re: new RegExp(`^/api/action-items/(${UUID})$`), kind: "action", type: "toggle" },
	{ verb: "delete", re: new RegExp(`^/api/action-items/(${UUID})$`), kind: "action", type: "delete" },
	{ verb: "post", re: /^\/api\/thankfuls\/?$/, kind: "thankful", type: "create" },
	{ verb: "delete", re: new RegExp(`^/api/thankfuls/(${UUID})$`), kind: "thankful", type: "delete" },
]

const LIST_SELECTOR = { action: "[action-item-list]", thankful: "[thankful-list]" }
const ITEM_SELECTOR = { action: "[action-item]", thankful: "[thankful]" }
const DELETE_LABEL = { action: "Delete action item", thankful: "Delete thankful" }

function param(parameters, name) {
	const value = parameters?.[name]
	return value == null ? "" : String(value)
}

function matchRoute(requestConfig) {
	const verb = String(requestConfig?.verb || "").toLowerCase()
	const path = requestConfig?.path || ""
	for (const route of ROUTES) {
		if (route.verb !== verb) continue
		const m = route.re.exec(path)
		if (m) return { route, serverId: m[1] }
	}
	return null
}

function pad(n) {
	return String(n).padStart(2, "0")
}

function stamp(date) {
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// A list item for something that exists only on this device so far. Built with
// DOM calls and textContent, never innerHTML: the text is the person's own.
function pendingItem(kind, op) {
	const li = document.createElement("li")
	li.className = "item"
	li.setAttribute(kind === "action" ? "action-item" : "thankful", "")
	li.dataset.clientId = op.ref.clientId

	const created = document.createElement("div")
	created.className = "created"
	const time = document.createElement("span")
	time.textContent = stamp(new Date(op.createdAt))
	const note = document.createElement("span")
	note.className = "pending-note"
	note.textContent = " · on this device"
	created.append(time, note)

	const info = document.createElement("div")
	info.className = "info"

	if (kind === "action") {
		const toggle = document.createElement("button")
		toggle.type = "button"
		toggle.className = "button"
		toggle.dataset.outboxToggle = ""
		toggle.setAttribute("aria-label", "Mark as complete")
		toggle.innerHTML = '<i class="fa fa-check" aria-hidden="true"></i>'
		info.append(toggle)
	}

	const text = document.createElement("p")
	text.className = op.isComplete ? "text completed" : "text"
	text.setAttribute("can-blur", "")
	text.setAttribute("can-reveal", "")
	text.tabIndex = 0
	text.textContent = op.text
	info.append(text)

	const del = document.createElement("button")
	del.type = "button"
	del.className = "button"
	del.dataset.outboxDelete = ""
	del.setAttribute("aria-label", DELETE_LABEL[kind])
	del.innerHTML = '<i class="fa fa-trash" aria-hidden="true"></i>'
	info.append(del)

	li.append(created, info)
	return li
}

function setCompleted(li, complete) {
	li.querySelector("p.text")?.classList.toggle("completed", complete)
	const flag = li.querySelector('input[name="isComplete"]')
	// The toggle form posts the state it would switch to.
	if (flag) flag.value = String(!complete)
	li.querySelector("button[aria-label^='Mark as']")?.setAttribute("aria-label", `Mark as ${complete ? "incomplete" : "complete"}`)
	li.dataset.completeLocal = String(complete)
}

function itemText(li) {
	return li.querySelector("p.text")?.firstChild?.textContent?.trim() || li.querySelector("p.text")?.textContent?.trim() || ""
}

function handleFailedRequest(event) {
	const found = matchRoute(event.detail?.requestConfig)
	if (!found) return
	const { route, serverId } = found
	const parameters = event.detail.requestConfig.parameters
	const elt = event.detail.requestConfig.elt

	if (route.type === "create") {
		// Linked to an entry or journal that only the server knows: can't be
		// replayed on its own.
		if (param(parameters, "journalId")) return
		const text = param(parameters, "text").trim()
		if (!text) return
		const op = enqueueOp({ kind: route.kind, type: "create", ref: {}, text })
		if (!op) return
		document.querySelector(LIST_SELECTOR[route.kind])?.append(pendingItem(route.kind, op))
		const input = document.getElementById(route.kind === "action" ? "text" : "thankful-text")
		if (input) input.value = ""
		return
	}

	const li = elt?.closest?.(ITEM_SELECTOR[route.kind])
	if (route.type === "delete") {
		if (!enqueueOp({ kind: route.kind, type: "delete", ref: { serverId } })) return
		li?.remove()
		return
	}

	// toggle
	const complete = param(parameters, "isComplete") === "true"
	const text = param(parameters, "text")
	if (!enqueueOp({
		kind: "action",
		type: complete ? "complete" : "uncomplete",
		ref: { serverId },
		text,
		completedAt: complete ? Date.now() : undefined,
	})) return
	if (li) setCompleted(li, complete)
}

document.addEventListener("htmx:sendError", handleFailedRequest)
document.addEventListener("htmx:timeout", handleFailedRequest)

// Controls on items that exist only on this device (the server-rendered ones
// are plain htmx forms and buttons).
document.addEventListener("click", (event) => {
	const toggle = event.target.closest?.("[data-outbox-toggle]")
	const del = event.target.closest?.("[data-outbox-delete]")
	if (!toggle && !del) return

	const li = event.target.closest("[data-client-id]")
	if (!li) return
	const kind = li.hasAttribute("action-item") ? "action" : "thankful"
	// Once synced, the client id is only good until the queue empties.
	const ref = li.dataset.serverId ? { serverId: li.dataset.serverId } : { clientId: li.dataset.clientId }

	if (del) {
		enqueueOp({ kind, type: "delete", ref })
		li.remove()
		return
	}
	const complete = li.dataset.completeLocal !== "true"
	enqueueOp({
		kind,
		type: complete ? "complete" : "uncomplete",
		ref,
		text: itemText(li),
		completedAt: complete ? Date.now() : undefined,
	})
	setCompleted(li, complete)
})

// A create reached the server: the item now has a real id, and nothing about
// it is waiting any more.
window.addEventListener("outbox:synced", (event) => {
	const { op, serverId } = event.detail || {}
	if (op?.type !== "create") return
	const li = document.querySelector(`[data-client-id="${CSS.escape(op.ref.clientId)}"]`)
	if (!li) return
	if (serverId) li.dataset.serverId = serverId
	li.querySelector(".pending-note")?.remove()
})

// The cached page predates whatever is still waiting in the outbox, so show
// those changes on it again after a load.
function showPending() {
	for (const op of pendingOps()) {
		if (op.rejected) continue
		if (op.type === "create") {
			const list = document.querySelector(LIST_SELECTOR[op.kind])
			if (!list || list.querySelector(`[data-client-id="${CSS.escape(op.ref.clientId)}"]`)) continue
			const li = pendingItem(op.kind, op)
			if (op.isComplete) setCompleted(li, true)
			list.append(li)
			continue
		}
		const id = op.ref.serverId
		if (!id) continue
		const li = document.getElementById(`${op.kind === "action" ? "action-item" : "thankful"}-${id}`)
		if (!li) continue
		if (op.type === "delete") li.remove()
		else setCompleted(li, op.type === "complete")
	}
}

document.addEventListener("DOMContentLoaded", showPending)
