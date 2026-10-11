// Styled replacement for window.confirm, backed by the shared
// <dialog id="confirm-dialog"> in base.templ. Resolves true on confirm,
// false on cancel/Esc/backdrop click. Falls back to the native prompt if the
// dialog isn't on the page.
export function confirmDialog(message, confirmLabel = "Confirm") {
	const dialog = document.getElementById("confirm-dialog")
	if (!dialog) return Promise.resolve(window.confirm(message))

	dialog.querySelector("[confirm-message]").textContent = message
	dialog.querySelector("[confirm-ok]").textContent = confirmLabel

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
