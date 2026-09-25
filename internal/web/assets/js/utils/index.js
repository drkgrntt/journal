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

// Checking an action item/thankful off, editing one, or adding a new one
// OOB-appends a hidden input into #action-item-ids/#thankful-ids (see
// list.templ's hx-swap-oob) so a journal entry submitted afterward picks it
// up. Both containers start out pre-seeded server-side with whatever this
// entry already has (empty for a brand-new entry) - snapshot that starting
// point on load so a reset can restore exactly it, instead of hardcoding
// "wipe to empty" (which would also erase an existing entry's real,
// already-saved associations).
const PENDING_CONTAINER_IDS = ["action-item-ids", "thankful-ids"]

function snapshotPendingContainers() {
	for (const id of PENDING_CONTAINER_IDS) {
		const container = document.getElementById(id)
		if (!container || container.dataset.pendingSnapshot != null) continue
		container.dataset.pendingSnapshot = container.innerHTML
	}
}
htmx.onLoad(snapshotPendingContainers)

const PAGE_LOAD_TIME = Date.now()
const STALE_QUEUE_THRESHOLD_MS = 30 * 60 * 1000

// A brand-new entry's containers start empty, so resetting means discarding
// anything queued before this draft started - but only once the queue has
// actually been sitting a while (this page staying open for a long stretch,
// e.g. items checked off hours earlier). Something checked off seconds
// before typing begins is presumably meant for this entry, so a fresh queue
// is left alone. An existing entry's containers start pre-seeded with its
// real, saved associations - there's no equivalent "session" there, so
// editing it always resets back to that saved set, discarding anything
// queued from interacting with unrelated items elsewhere on the page (e.g.
// the Outstanding sidebar on an old entry you're just reviewing). Only fires
// once per draft (see draftStarted guard) so later keystrokes don't reset
// items checked mid-write.
export function resetPendingActionItemIds(event) {
	const textarea = event.target
	if (textarea.dataset.draftStarted) return
	textarea.dataset.draftStarted = "true"

	const isExistingEntry = textarea.form && textarea.form.dataset.existing === "true"
	if (!isExistingEntry && Date.now() - PAGE_LOAD_TIME < STALE_QUEUE_THRESHOLD_MS) return

	for (const id of PENDING_CONTAINER_IDS) {
		const container = document.getElementById(id)
		if (container) container.innerHTML = container.dataset.pendingSnapshot ?? ""
	}
}
setToWindow("resetPendingActionItemIds", resetPendingActionItemIds)

// Toggling the same action item/thankful complete and back queues it again -
// list.templ's OOB append has no memory of what's already queued, so this
// can leave duplicate hidden inputs with the same value. Harmless to the
// save itself (duplicate ids just resolve to the same row), but clean it up
// after every htmx swap so the queue reflects what's actually checked.
function dedupePendingContainers() {
	for (const id of PENDING_CONTAINER_IDS) {
		const container = document.getElementById(id)
		if (!container) continue
		const seen = new Set()
		const inputs = [...container.querySelectorAll("input[type=hidden]")].reverse()
		for (const input of inputs) {
			if (seen.has(input.value)) {
				input.remove()
			} else {
				seen.add(input.value)
			}
		}
	}
}
document.body.addEventListener("htmx:afterSettle", dedupePendingContainers)

;(function() {
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

// Shared by the dashboard's rating charts (mood-by-day/tod/topic, the
// month-view mood chart): Math.min(...[]) / Math.max(...[]) evaluate to
// +/-Infinity for an empty values array, which produces an inverted/broken
// y-axis instead of degrading gracefully. Returns null when there's nothing
// to chart so the caller can render an empty state instead of handing
// Chart.js a broken range; otherwise the padded [min, max] range for the
// axis (round=false skips the floor/ceil, for charts - like the month view -
// whose values aren't already whole rating numbers being padded to whole
// ticks).
export function ratingAxisRange(values, { round = true } = {}) {
	if (!values.length) return null
	const lo = Math.min(...values)
	const hi = Math.max(...values)
	return {
		min: (round ? Math.floor(lo) : lo) - 0.3,
		max: (round ? Math.ceil(hi) : hi) + 0.3,
	}
}
setToWindow("ratingAxisRange", ratingAxisRange)

// Paired with ratingAxisRange: swaps a chart container's contents for a
// simple "no data" message instead of building a chart out of an empty
// dataset.
export function renderChartEmptyState(element, message) {
	element.innerHTML = ""
	const p = document.createElement("p")
	p.className = "chart-empty-state"
	p.textContent = message || "No data for this range yet."
	element.appendChild(p)
}
setToWindow("renderChartEmptyState", renderChartEmptyState)

// Builds a { [rating.value]: rating.name } lookup so chart y-axis tick
// labels reflect the actual Rating table instead of a hardcoded 5-point
// scale that can silently drift from it.
export function ratingLabelsByValue(ratings) {
	const labels = {}
	for (const rating of ratings || []) {
		labels[rating.value] = rating.name
	}
	return labels
}
setToWindow("ratingLabelsByValue", ratingLabelsByValue)

// Shared by the dashboard's rating charts (mood-by-day/tod/topic, the
// month-view mood chart): these all build a single-dataset bar/line Chart.js
// chart out of a `range` (from ratingAxisRange) and `ratingLabels` (from
// ratingLabelsByValue), styled/gridded/legend-less the same way, and mount
// it into a container element the same way (clear it, append a fresh
// canvas). This factors that construction out; callers still own their own
// chart-data mapping (e.g. converting a time-of-day string, filtering
// zero/empty values before computing `range`), which Chart.js `type` they
// need, dataset label/colors, and any extra top-level or per-dataset options
// specific to that chart (e.g. `tension`, `spanGaps`).
export function renderRatingChart(element, {
	type,
	labels,
	values,
	range,
	ratingLabels,
	datasetLabel = "Average Rating",
	borderColor,
	backgroundColor,
	borderWidth,
	dataset = {},
	options = {},
} = {}) {
	const canvas = document.createElement("canvas")
	const ctx = canvas.getContext("2d")
	if (!ctx) {
		throw new Error("Canvas not supported")
	}

	const chart = new Chart(ctx, {
		type,
		data: {
			labels,
			datasets: [
				{
					label: datasetLabel,
					data: values,
					borderColor,
					backgroundColor,
					...(borderWidth !== undefined ? { borderWidth } : {}),
					...dataset,
				},
			],
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			...options,
			plugins: {
				legend: {
					display: false,
				},
				...(options.plugins || {}),
			},
			scales: {
				x: {
					grid: {
						display: false,
					},
				},
				y: {
					min: range.min,
					max: range.max,
					grid: {
						display: false,
					},
					// beginAtZero: true,
					ticks: {
						callback: function(value) {
							if (value % 1 !== 0) return;
							return ratingLabels[value] ?? value;
						}
					},
				},
				...(options.scales || {}),
			},
		},
	})

	element.innerHTML = "";
	element.appendChild(canvas);

	return chart;
}
setToWindow("renderRatingChart", renderRatingChart)

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
