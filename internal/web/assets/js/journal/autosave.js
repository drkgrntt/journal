import { currentUserId } from "../utils/index.js"
import { loadDraft, markDraftActive, newDraftId, removeDraft, saveDraft, sendEntry } from "../utils/drafts.js"

const AUTOSAVE_DELAY_MS = 1500
const RETRY_DELAY_MS = 30 * 1000
const SAVED_STATUS_DISPLAY_MS = 3000

function initAutosave() {
	const form = document.getElementById("journal-form")
	if (!form) return
	if (form.dataset.autosaveInit) return
	form.dataset.autosaveInit = "true"

	const statusEl = form.querySelector("#autosave-status")
	const entryEl = form.querySelector("#entry")

	let debounceTimer = null
	let statusTimer = null
	let retryTimer = null
	let inFlight = false
	let dirty = false
	// Set while the last save couldn't reach the server (offline, 5xx, expired
	// session); the text is safe in an on-device draft until it can.
	let needsRetry = false

	// A brand-new entry gets a client-generated id, sent with every save, so
	// an on-device draft can be retried without ever creating the entry twice
	// (the server recognizes the id - see createJournal). Existing entries
	// aren't drafted: they save straight to the server.
	const isNewEntry = form.dataset.existing !== "true" && form.hasAttribute("hx-post")
	const draftId = isNewEntry ? newDraftId() : null
	if (draftId) {
		const hidden = document.createElement("input")
		hidden.type = "hidden"
		hidden.name = "draftId"
		hidden.value = draftId
		form.appendChild(hidden)
		form.dataset.draftId = draftId
		markDraftActive(draftId, form)
	}

	function setStatus(text) {
		clearTimeout(statusTimer)
		if (statusEl) statusEl.textContent = text
	}

	function showSavedStatus() {
		setStatus("Saved!")
		statusTimer = setTimeout(() => {
			const time = new Date().toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })
			if (statusEl) statusEl.textContent = `Saved at ${time}`
		}, SAVED_STATUS_DISPLAY_MS)
	}

	function currentId() {
		const put = form.getAttribute("hx-put")
		if (!put) return null
		return put.split("/").pop()
	}

	function crossfadeText(el, text) {
		if (!el) return
		el.style.transition = "opacity var(--duration-fast) var(--ease-standard)"
		el.style.opacity = "0"
		el.addEventListener(
			"transitionend",
			() => {
				el.textContent = text
				el.style.opacity = "1"
			},
			{ once: true },
		)
	}

	function updatePageChrome(id) {
		// The New Entry page becomes the Edit Entry page in place - no
		// reload, so nothing steals focus from the textarea mid-sentence.
		const page = document.getElementById("new-journal-page")
		if (!page) return
		page.id = "edit-journal-page"

		const backLink = page.querySelector(":scope > a.back")
		if (backLink) backLink.setAttribute("href", `/journal/${id}`)

		crossfadeText(page.querySelector(":scope > h1.heading"), "Edit Journal Entry")
	}

	function promoteFormToEditMode(id) {
		form.removeAttribute("hx-post")
		form.setAttribute("hx-put", `/api/journal/${id}`)
		htmx.process(form)
		history.replaceState(null, "", `/journal/${id}/edit`)
		updatePageChrome(id)
	}

	function snapshotBody() {
		const formData = new FormData(form)
		formData.set("autosave", "true")
		// url-encoded, not FormData's default multipart - see below.
		return new URLSearchParams(formData).toString()
	}

	// Written on every input, ahead of the debounced server save, so even a
	// tab closed mid-sentence leaves the text behind.
	function persistDraft() {
		if (!draftId) return
		const id = currentId()
		if (!id && (!entryEl || entryEl.value.trim() === "")) {
			removeDraft(currentUserId(), draftId)
			return
		}
		saveDraft({ id: draftId, userId: currentUserId(), serverId: id, body: snapshotBody() })
	}

	function scheduleRetry() {
		clearTimeout(retryTimer)
		retryTimer = setTimeout(retryIfNeeded, RETRY_DELAY_MS)
	}

	function retryIfNeeded() {
		if (!form.isConnected) return
		if (needsRetry && !inFlight && document.visibilityState === "visible") runSave()
		else if (needsRetry) scheduleRetry()
	}

	function runSave() {
		if (inFlight) {
			dirty = true
			return
		}

		const id = currentId()
		const entryEmpty = !entryEl || entryEl.value.trim() === ""
		if (!id && entryEmpty) return

		// FormData's default multipart/form-data encoding doesn't get the
		// actionItemIds[]/thankfulIds[] bracket-stripping the server only
		// applies to url-encoded bodies, so those associations would
		// silently never update via autosave. Send url-encoded instead,
		// matching what the manual "Done" submit already sends via htmx.
		const body = snapshotBody()
		persistDraft()

		inFlight = true
		setStatus("Saving…")

		sendEntry(id, body)
			.then((result) => {
				if (result.status === "saved") {
					needsRetry = false
					if (!id) promoteFormToEditMode(result.entry.id)
					settleDraft(body, result.entry.id)
					showSavedStatus()
				} else if (result.status === "rejected") {
					needsRetry = false
					setStatus("Error saving")
				} else {
					needsRetry = true
					setStatus(
						result.status === "auth"
							? "Saved on this device. Log in to sync"
							: "Saved on this device. Will sync when you're back online",
					)
					scheduleRetry()
				}
			})
			.finally(() => {
				inFlight = false
				if (dirty) {
					dirty = false
					runSave()
				}
			})
	}

	// The server has `body`. Drop the on-device copy unless typing has moved
	// past it in the meantime, in which case keep it (now pointing at the saved
	// entry) for the save that follows.
	function settleDraft(body, serverId) {
		if (!draftId) return
		const userId = currentUserId()
		const draft = loadDraft(userId, draftId)
		if (!draft || draft.body === body) {
			removeDraft(userId, draftId)
		} else {
			saveDraft({ ...draft, serverId })
		}
	}

	function scheduleSave() {
		clearTimeout(debounceTimer)
		debounceTimer = setTimeout(runSave, AUTOSAVE_DELAY_MS)
	}

	function onEdit() {
		persistDraft()
		scheduleSave()
	}

	form.addEventListener("input", onEdit)
	form.addEventListener("change", onEdit)
	form.addEventListener("submit", () => clearTimeout(debounceTimer))

	// "Done" goes through htmx rather than autosave. Once the server has it
	// the draft is redundant; if the request never got there, the draft stays
	// and the form keeps retrying like any other save.
	form.addEventListener("htmx:afterRequest", (event) => {
		if (event.target !== form || !draftId) return
		if (event.detail.successful) removeDraft(currentUserId(), draftId)
	})
	form.addEventListener("htmx:sendError", (event) => {
		if (event.target !== form) return
		persistDraft()
		needsRetry = true
		setStatus("Saved on this device. Will sync when you're back online")
		scheduleRetry()
	})

	window.addEventListener("online", retryIfNeeded)
	document.addEventListener("visibilitychange", retryIfNeeded)
}

htmx.onLoad(initAutosave)
