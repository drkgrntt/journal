import { makeRatingColorizer } from '/assets/js/utils/ratingColors.js'

function initYearInPixels() {
  const root = document.querySelector(".year-in-pixels")
  if (!root) return
  const days = JSON.parse(document.getElementById("year-in-pixels-days").textContent) || []
  const ratings = JSON.parse(document.getElementById("year-in-pixels-ratings").textContent) || []
  const colorForRating = makeRatingColorizer(ratings)
  const maxValue = ratings.length ? Math.max(...ratings.map(r => r.value)) : 5

  const byDate = {}
  days.forEach(function(day) { byDate[day.date] = day })

  root.querySelectorAll("[data-year-date]").forEach(function(el) {
    const dateKey = el.dataset.yearDate
    const day = byDate[dateKey]
    if (!day) return

    const [year, month, d] = dateKey.split('-').map(Number)
    const dateString = new Date(year, month - 1, d).toLocaleDateString(undefined, {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    })

    const link = document.createElement('a')
    link.className = 'pixel rated'
    link.href = `/journal?date=${dateKey}`
    link.title = `${dateString}\nRating: ${day.average.toFixed(1)} / ${maxValue}\nFrom ${day.count} journal(s)`
    link.style.color = colorForRating(day.average)
    link.textContent = '●'
    el.replaceWith(link)
  })
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-year-in-pixels", initYearInPixels)
document.addEventListener("load-year-in-pixels", initYearInPixels)
