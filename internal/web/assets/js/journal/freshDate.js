// A new-entry form left open overnight would otherwise keep the date it loaded
// with. When the tab becomes visible again, refresh the date to "now" - but only
// while the entry is still genuinely new: nothing typed yet (so no autosave has
// promoted the form to edit mode, which swaps hx-post for hx-put) and the date
// was never touched by hand. Once someone starts writing, the date stays at when
// they started.
function initFreshDate() {
	const form = document.getElementById("journal-form")
	if (!form) return
	if (form.dataset.freshDateInit) return
	form.dataset.freshDateInit = "true"

	const dateEl = form.querySelector("#date")
	const unixEl = dateEl?.nextElementSibling
	const entryEl = form.querySelector("#entry")
	if (!dateEl || !unixEl || !entryEl) return

	let touched = false
	dateEl.addEventListener("input", () => (touched = true))
	dateEl.addEventListener("change", () => (touched = true))

	function pad(n) {
		return String(n).padStart(2, "0")
	}

	function refreshDate() {
		if (document.visibilityState !== "visible") return
		if (!form.isConnected) {
			document.removeEventListener("visibilitychange", refreshDate)
			window.removeEventListener("focus", refreshDate)
			return
		}
		if (touched) return
		if (!form.hasAttribute("hx-post")) return
		if (entryEl.value.trim() !== "") return

		const now = new Date()
		dateEl.value = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}T${pad(now.getHours())}:${pad(now.getMinutes())}`
		unixEl.value = new Date(dateEl.value).getTime()
	}

	document.addEventListener("visibilitychange", refreshDate)
	window.addEventListener("focus", refreshDate)
}

htmx.onLoad(initFreshDate)
