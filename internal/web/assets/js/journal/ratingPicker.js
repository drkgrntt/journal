function initRatingPicker() {
	const container = document.querySelector(".rating-picker")
	if (!container) return
	if (container.dataset.ratingPickerInit) return
	container.dataset.ratingPickerInit = "true"

	// Native radios can't be unchecked by clicking the one that's already
	// selected - track the currently selected one ourselves so a repeat
	// click can clear it, letting a journal go back to having no rating.
	let selected = container.querySelector('input[name="rating"]:checked')

	container.addEventListener("click", function (event) {
		const input = event.target.closest('input[name="rating"]')
		if (!input) return

		if (input === selected) {
			input.checked = false
			selected = null
		} else {
			selected = input
		}
	})
}

htmx.onLoad(initRatingPicker)
