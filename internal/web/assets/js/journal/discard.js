function initDiscard() {
	const form = document.getElementById("journal-form")
	if (!form) return

	const discardBtn = document.getElementById("discard-button")
	if (!discardBtn) return
	if (discardBtn.dataset.discardInit) return
	discardBtn.dataset.discardInit = "true"

	// An entry that already existed when the page loaded gets reverted to
	// these original values rather than deleted - snapshot them now, before
	// autosave or the user can change anything.
	const isExisting = form.dataset.existing === "true"
	// url-encoded, not FormData's default multipart, so the reverted
	// actionItemIds[]/thankfulIds[] fields actually get parsed - see
	// autosave.js for the same fix and why.
	const originalData = isExisting ? new URLSearchParams(new FormData(form)) : null

	function currentId() {
		const put = form.getAttribute("hx-put")
		if (!put) return null
		return put.split("/").pop()
	}

	function redirectFrom(res) {
		if (!res.ok) {
			alert("Error discarding entry")
			return
		}
		window.location.href = res.headers.get("HX-Redirect") || "/journal"
	}

	discardBtn.addEventListener("click", function (event) {
		event.preventDefault()

		const id = currentId()

		if (isExisting) {
			if (!confirm("Discard changes and revert to the last saved version?")) return
			fetch(`/api/journal/${id}`, {
				method: "PUT",
				body: originalData,
				credentials: "same-origin",
			}).then(redirectFrom)
			return
		}

		if (!id) {
			window.location.href = "/journal"
			return
		}

		if (!confirm("Discard this entry?")) return
		fetch(`/api/journal/${id}`, {
			method: "DELETE",
			credentials: "same-origin",
		}).then(redirectFrom)
	})
}

htmx.onLoad(initDiscard)
