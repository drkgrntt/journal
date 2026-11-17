// "Start from a template" on the new-entry form. The server renders only an
// empty, hidden picker for subscribers; the templates themselves are fetched
// here at runtime, so the service worker's cached /journal/new page never holds
// template text. Offline or on any error the picker simply stays hidden.

function initTemplatePicker() {
	const picker = document.getElementById("template-picker")
	if (!picker || picker.dataset.templatePickerInit) return
	picker.dataset.templatePickerInit = "true"

	const select = picker.querySelector("select")
	const entryEl = document.getElementById("entry")
	if (!select || !entryEl) return

	let templates = []

	fetch("/api/entry-templates", { headers: { Accept: "application/json" }, credentials: "same-origin" })
		.then((response) => (response.ok ? response.json() : []))
		.then((items) => {
			if (!Array.isArray(items) || items.length === 0) return
			templates = items
			for (const item of items) {
				const option = document.createElement("option")
				option.value = item.id
				option.textContent = item.name
				select.appendChild(option)
			}
			picker.hidden = false
		})
		.catch(() => {})

	select.addEventListener("change", async () => {
		const template = templates.find((item) => item.id === select.value)
		if (!template) return

		if (entryEl.value.trim() !== "" && !(await window.confirmDialog("Replace what you've written with this template?", "Replace"))) {
			select.value = ""
			return
		}

		entryEl.value = template.body
		// Go through the normal input event so autosave writes the on-device
		// draft and schedules the server save like any typing would.
		entryEl.dispatchEvent(new Event("input", { bubbles: true }))
		entryEl.focus()
		entryEl.setSelectionRange(entryEl.value.length, entryEl.value.length)
		select.value = ""
	})
}

htmx.onLoad(initTemplatePicker)
