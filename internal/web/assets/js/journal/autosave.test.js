// Run: node --test internal/web/assets/js/journal/autosave.test.js
//
// Drives the real autosave.js against a small stand-in for the page: just the
// globals and the form behavior it touches.
import { test, beforeEach } from "node:test"
import assert from "node:assert/strict"

const store = new Map()
globalThis.localStorage = {
	getItem: (k) => store.get(k) ?? null,
	setItem: (k, v) => store.set(k, String(v)),
	removeItem: (k) => store.delete(k),
	key: (i) => [...store.keys()][i] ?? null,
	get length() {
		return store.size
	},
}

let page = null
const onLoad = []
globalThis.window = new EventTarget()
globalThis.document = Object.assign(new EventTarget(), {
	body: new EventTarget(),
	visibilityState: "visible",
	getElementById: (id) => (id === "journal-form" ? page : null),
	querySelector: (sel) => (sel === "[data-user-id]" ? { dataset: { userId: "u1" } } : null),
	querySelectorAll: () => [],
	createElement: () => ({}),
})
globalThis.htmx = { onLoad: (fn) => onLoad.push(fn), process: () => {} }
globalThis.history = { replaceState: () => {} }
// new FormData(form) only needs to yield the form's current fields.
globalThis.FormData = class extends URLSearchParams {
	constructor(form) {
		super(form.fields())
	}
}

let sent = []
globalThis.fetch = async (url, init) => {
	sent.push({ url, body: new URLSearchParams(init.body) })
	return { ok: true, status: 201, type: "basic", json: async () => ({ id: "server-1" }) }
}

await import("./autosave.js")
const initAutosave = onLoad.at(-1)

class FakeForm extends EventTarget {
	dataset = {}
	isConnected = true
	entry = { value: "" }
	rating = ""
	status = { textContent: "", style: {} }
	attributes = new Map([["hx-post", "/api/journal"]])
	hidden = []
	hasAttribute(name) {
		return this.attributes.has(name)
	}
	getAttribute(name) {
		return this.attributes.get(name) ?? null
	}
	setAttribute(name, value) {
		this.attributes.set(name, value)
	}
	removeAttribute(name) {
		this.attributes.delete(name)
	}
	querySelector(sel) {
		if (sel === "#entry") return this.entry
		if (sel === "#autosave-status") return this.status
		return null
	}
	appendChild(el) {
		this.hidden.push(el)
	}
	fields() {
		const fields = { entry: this.entry.value, rating: this.rating }
		for (const el of this.hidden) fields[el.name] = el.value
		return fields
	}
}

function newForm({ rating = "", entry = "" } = {}) {
	const form = new FakeForm()
	form.rating = rating
	form.entry.value = entry
	page = form
	initAutosave()
	return form
}

const emit = (target, type, detail) => target.dispatchEvent(Object.assign(new Event(type), { detail }))
const drafts = () => [...store.values()].map((v) => new URLSearchParams(JSON.parse(v).body))
const settle = () => new Promise((resolve) => setImmediate(resolve))

// "Done" while offline: htmx submits, then reports it couldn't send.
function submitOffline(form) {
	emit(form, "submit")
	emit(form, "htmx:sendError", {})
}

beforeEach(() => {
	// Earlier tests' forms leave the page, so they stop retrying.
	if (page) page.isConnected = false
	store.clear()
	sent = []
})

test("a rating-only entry submitted offline is kept on the device", () => {
	const form = newForm({ rating: "4" })
	emit(form, "change")
	submitOffline(form)

	assert.equal(drafts().length, 1)
	assert.equal(drafts()[0].get("rating"), "4")
})

test("a rating-only entry submitted offline is sent when back online", async () => {
	const form = newForm({ rating: "4" })
	submitOffline(form)
	emit(window, "online")
	await settle()

	assert.equal(sent.length, 1)
	assert.equal(sent[0].url, "/api/journal")
	assert.equal(sent[0].body.get("rating"), "4")
	assert.ok(sent[0].body.get("draftId"))
	assert.equal(drafts().length, 0)
})

test("a rating-only submit the server fails on is kept", () => {
	const form = newForm({ rating: "5" })
	emit(form, "submit")
	emit(form, "htmx:afterRequest", { successful: false })

	assert.equal(drafts().length, 1)
})

test("a rating-only submit the server accepts leaves no draft", () => {
	const form = newForm({ rating: "5" })
	emit(form, "submit")
	emit(form, "htmx:afterRequest", { successful: true })

	assert.equal(drafts().length, 0)
})

test("picking a rating without submitting leaves no draft", () => {
	const form = newForm({ rating: "3" })
	emit(form, "change")

	assert.equal(drafts().length, 0)
})

test("erasing typed text without submitting removes the draft", () => {
	const form = newForm({ entry: "hello" })
	emit(form, "input")
	assert.equal(drafts().length, 1)

	form.entry.value = ""
	emit(form, "input")
	assert.equal(drafts().length, 0)
})
