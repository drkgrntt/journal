import { makeRatingColorizer } from '/assets/js/utils/ratingColors.js'

function initRatingDistribution() {
  const element = document.querySelector(".rating-distribution")
  if (!element) return
  const data = JSON.parse(document.getElementById("rating-distribution-data").textContent) || []
  const ratings = JSON.parse(document.getElementById("rating-distribution-ratings-data").textContent) || []
  const names = ratingLabelsByValue(ratings)
  const colorForRating = makeRatingColorizer(ratings)
  const total = data.reduce((sum, item) => sum + item.count, 0)

  if (total === 0) {
    renderChartEmptyState(element)
    return
  }

  const canvas = document.createElement("canvas")
  const ctx = canvas.getContext("2d")
  if (!ctx) {
    throw new Error("Canvas not supported")
  }

  const colors = data.map(item => colorForRating(item.value))

  new Chart(ctx, {
    type: "bar",
    data: {
      labels: data.map(item => names[item.value] || String(item.value)),
      datasets: [
        {
          label: "Entries",
          data: data.map(item => item.count),
          backgroundColor: colors,
          borderColor: colors,
          borderWidth: 2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      minBarLength: 2,
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: function(context) {
              const item = data[context.dataIndex]
              const noun = item.count === 1 ? "entry" : "entries"
              return `${item.count} ${noun} (${Math.round(item.percent)}%)`
            },
          },
        },
      },
      scales: {
        x: { grid: { display: false } },
        y: {
          beginAtZero: true,
          grid: { display: false },
          ticks: { precision: 0 },
        },
      },
    },
  })

  element.innerHTML = ""
  element.appendChild(canvas)
}
// Remove before add: prevents duplicate listeners from stacking up on
// `document` when this fragment (and its <script> tag) is swapped in again.
document.removeEventListener("load-rating-distribution", initRatingDistribution)
document.addEventListener("load-rating-distribution", initRatingDistribution)
