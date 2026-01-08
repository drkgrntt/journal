(function() {
  const journals = JSON.parse(document.getElementById("mood-chart-journals").textContent);
  const element = document.getElementById("mood-chart");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const sortedJournals = journals
    .sort(function(a, b) { return new Date(a.date) - new Date(b.date) })

  const mapping = {}
  const dates = Array.from(new Set(sortedJournals.map(function(journal) {
    const date = new Date(journal.date).toLocaleDateString();
    mapping[date] ||= [];
    mapping[date].push({
      rating: journal.rating.value,
    });
    return date;
  })))

  const ratingData = dates.map(function(date) {
    const rate = mapping[date].reduce(function(acc, { rating }) {
      return acc + rating;
    }, 0);
    return rate / mapping[date].length;
  });

  const chart = new Chart(ctx, {
    type: "line",
    data: {
      labels: dates,
      datasets: [
        {
          label: "Rating",
          data: ratingData,
          borderColor: getCssValue("--primary-color-dark"),
          backgroundColor: getCssValue("--primary-color"),
        },
      ],
    },
    options: {
      tension: .4,
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          grid: {
            display: false,
          },
        },
        y: {
          grid: {
            display: false,
          },
          // beginAtZero: true,
          ticks: {
            callback: function(value) {
              if (value % 1 !== 0) return;
              if (value > 5) return value;
              return [
                "Awful",
                "Bad",
                "Fine",
                "Good",
                "Great",
              ][value - 1]
            }
          }
        }
      },
    },
  })

  // Show the chart
  element.innerHTML = "";
  element.appendChild(canvas);
})();
