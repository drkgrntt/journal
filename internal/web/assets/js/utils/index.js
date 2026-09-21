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
