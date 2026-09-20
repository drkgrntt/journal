export * from "./dates.js"

export function setToWindow(key, item) {
	window[key] = item
}

export function hideElement(event, element) {
	if (!event.detail.failed) {
		if (!element) return
		element.classList.add("util-hidden")
	}
}
setToWindow("hideElement", hideElement)

export function showElement(event, element) {
	if (!event.detail.failed) {
		if (!element) return
		element.classList.remove("util-hidden")
	}
}
setToWindow("showElement", showElement)

export function handleHtmxError(event) {
	if (event.detail.failed) {
		alert(event.detail.xhr.response)
	}
}
setToWindow("handleHtmxError", handleHtmxError)

export function handleHtmxSuccess(event, message) {
	if (!event.detail.failed) {
		alert(message)
	}
}
setToWindow("handleHtmxSuccess", handleHtmxSuccess)

export function setUnixValue(dateString, element) {
	if (!element) return
	const date = new Date(dateString)
	element.value = date.getTime()
}
setToWindow("setUnixValue", setUnixValue);

// Checking an action item OOB-appends it into #action-item-ids so a journal
// entry submitted afterward picks it up. That queue otherwise survives for as
// long as the dashboard page stays open, so a box checked hours ago still
// attaches to an unrelated entry written later. Starting to type a brand-new
// entry is treated as the start of a fresh session: it wipes anything queued
// before that point, so only items checked while this entry is actively being
// written still attach. Only fires once per draft (see draftStarted guard) so
// later keystrokes don't wipe items checked mid-write.
export function resetPendingActionItemIds(event) {
	const textarea = event.target
	if (textarea.dataset.draftStarted) return
	textarea.dataset.draftStarted = "true"
	const container = document.getElementById("action-item-ids")
	if (container) container.innerHTML = ""
}
setToWindow("resetPendingActionItemIds", resetPendingActionItemIds)

(function() {
	const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
	document.cookie = `tz=${tz}; Path=/; Max-Age=31536000`
})()


export function getCssValue(variable) {
	const style = window.getComputedStyle(document.body)
	return style.getPropertyValue(variable)
}
setToWindow("getCssValue", getCssValue)

export function shouldBlurText() {
	return localStorage.getItem(`blur-text-${document.body.id}`) === "true"
}
setToWindow("shouldBlurText", shouldBlurText)

function isEmptyField(element) {
	if (element.tagName !== "INPUT" && element.tagName !== "TEXTAREA") return false
	return element.value === ""
}

function applyBlur(element) {
	const blur = shouldBlurText() && !isEmptyField(element)
	element.style.filter = blur ? "blur(3px)" : "none"
}

htmx.onLoad(function(_e) {
	document.querySelectorAll("[can-blur]").forEach(applyBlur)
})

// Elements are blurred by default via CSS to avoid flashing sensitive text
// before the setting above resolves; inputs/textareas additionally reveal
// on focus so they stay editable, then re-blur once focus leaves.
document.addEventListener("focusin", function(e) {
	const element = e.target.closest("[can-blur]")
	if (!element) return
	if (element.tagName !== "INPUT" && element.tagName !== "TEXTAREA") return
	element.style.filter = "none"
})

document.addEventListener("focusout", function(e) {
	const element = e.target.closest("[can-blur]")
	if (!element) return
	if (element.tagName !== "INPUT" && element.tagName !== "TEXTAREA") return
	applyBlur(element)
})
