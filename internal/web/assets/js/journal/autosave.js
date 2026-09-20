const AUTOSAVE_DELAY_MS = 1500

function initAutosave() {
	const form = document.getElementById("journal-form")
	if (!form) return
	if (form.dataset.autosaveInit) return
	form.dataset.autosaveInit = "true"

	const statusEl = form.querySelector("#autosave-status")
	const entryEl = form.querySelector("#entry")

	let debounceTimer = null
	let inFlight = false
	let dirty = false

	function setStatus(text) {
		if (statusEl) statusEl.textContent = text
	}

	function currentId() {
		const put = form.getAttribute("hx-put")
		if (!put) return null
		return put.split("/").pop()
	}

	function hasRatingSelected() {
		return !!form.querySelector('input[name="rating"]:checked')
	}

	function promoteFormToEditMode(id) {
		form.removeAttribute("hx-post")
		form.setAttribute("hx-put", `/api/journal/${id}`)
		htmx.process(form)
		history.replaceState(null, "", `/journal/${id}/edit`)
	}

	function runSave() {
		if (inFlight) {
			dirty = true
			return
		}

		const id = currentId()
		const entryEmpty = !entryEl || entryEl.value.trim() === ""
		if (!id && (entryEmpty || !hasRatingSelected())) return

		const formData = new FormData(form)
		formData.set("autosave", "true")

		const url = id ? `/api/journal/${id}` : "/api/journal"
		const method = id ? "PUT" : "POST"

		inFlight = true
		setStatus("Saving…")

		fetch(url, { method, body: formData, credentials: "same-origin" })
			.then(async (res) => {
				if (!res.ok) throw new Error(await res.text())
				const journal = await res.json()
				if (!id) promoteFormToEditMode(journal.id)
				setStatus("Saved")
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
