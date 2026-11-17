// Styled replacement for window.confirm, backed by the shared
// <dialog id="confirm-dialog"> in base.templ. Resolves true on confirm,
// false on cancel/Esc/backdrop click. Falls back to the native prompt if the
// dialog isn't on the page.
export function confirmDialog(message, confirmLabel = "Confirm") {
	const dialog = document.getElementById("confirm-dialog")
	if (!dialog) return Promise.resolve(window.confirm(message))

	dialog.querySelector("[confirm-message]").textContent = message
	dialog.querySelector("[confirm-ok]").textContent = confirmLabel
	dialog.querySelector("[confirm-cancel]").hidden = false

	return new Promise((resolve) => {
		dialog.addEventListener(
			"close",
			() => resolve(dialog.returnValue === "ok"),
			{ once: true },
		)
		dialog.returnValue = ""
		dialog.showModal()
		dialog.querySelector("[confirm-cancel]").focus()
	})
}
window.confirmDialog = confirmDialog

// Styled replacement for window.alert: the same dialog with just an OK button.
// Resolves once it's dismissed.
export function alertDialog(message) {
	const dialog = document.getElementById("confirm-dialog")
	if (!dialog) {
		window.alert(message)
		return Promise.resolve()
	}

	dialog.querySelector("[confirm-message]").textContent = message
	dialog.querySelector("[confirm-ok]").textContent = "OK"
	const cancel = dialog.querySelector("[confirm-cancel]")
	cancel.hidden = true

	return new Promise((resolve) => {
		dialog.addEventListener("close", () => resolve(), { once: true })
		dialog.returnValue = ""
		dialog.showModal()
		dialog.querySelector("[confirm-ok]").focus()
	})
}
window.alertDialog = alertDialog

document.addEventListener("click", (event) => {
	const dialog = event.target
	if (!(dialog instanceof HTMLDialogElement) || dialog.id !== "confirm-dialog") return
	// A click on the dialog element itself (not its content) is the backdrop.
	dialog.close("cancel")
})

// Route every hx-confirm through the dialog instead of the native prompt.
document.body.addEventListener("htmx:confirm", (event) => {
	if (!event.detail.question) return
	event.preventDefault()
	confirmDialog(event.detail.question, "Yes").then((ok) => {
		if (ok) event.detail.issueRequest(true)
	})
})
