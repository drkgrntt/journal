const AUTOSAVE_DELAY_MS = 1500
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
	let inFlight = false
	let dirty = false

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

	function runSave() {
		if (inFlight) {
			dirty = true
			return
		}

		const id = currentId()
		const entryEmpty = !entryEl || entryEl.value.trim() === ""
		if (!id && entryEmpty) return

		const formData = new FormData(form)
		formData.set("autosave", "true")
		// FormData's default multipart/form-data encoding doesn't get the
		// actionItemIds[]/thankfulIds[] bracket-stripping the server only
		// applies to url-encoded bodies, so those associations would
		// silently never update via autosave. Send url-encoded instead,
		// matching what the manual "Done" submit already sends via htmx.
		const params = new URLSearchParams(formData)

		const url = id ? `/api/journal/${id}` : "/api/journal"
		const method = id ? "PUT" : "POST"

		inFlight = true
		setStatus("Saving…")

		fetch(url, { method, body: params, credentials: "same-origin" })
			.then(async (res) => {
				if (!res.ok) throw new Error(await res.text())
				const journal = await res.json()
				if (!id) promoteFormToEditMode(journal.id)
				showSavedStatus()
			})
			.catch(() => setStatus("Error saving"))
			.finally(() => {
				inFlight = false
				if (dirty) {
					dirty = false
					runSave()
				}
			})
	}

	function scheduleSave() {
		clearTimeout(debounceTimer)
		debounceTimer = setTimeout(runSave, AUTOSAVE_DELAY_MS)
	}

	form.addEventListener("input", scheduleSave)
	form.addEventListener("change", scheduleSave)
	form.addEventListener("submit", () => clearTimeout(debounceTimer))
}

htmx.onLoad(initAutosave)
