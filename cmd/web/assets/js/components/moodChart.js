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

  const dateRateMapping = {}
  const dates = Array.from(new Set(sortedJournals.map(function(journal) {
    const date = new Date(journal.date).toLocaleDateString();
    dateRateMapping[date] ||= [];
    dateRateMapping[date].push(journal.rating.value);
    return date;
  })))

  const chartData = dates.map(function(date) {
    const rate = dateRateMapping[date].reduce(function(acc, rating) {
      return acc + rating;
    }, 0);
    return rate / dateRateMapping[date].length;
  });

  console.log({ dates, chartData })

  var style = window.getComputedStyle(document.body)

  const chart = new Chart(ctx, {
    type: "line",
    data: {
      labels: dates,
      datasets: [
        {
          tension: .4,
          label: "Rating",
          data: chartData,
          borderColor: style.getPropertyValue("--primary-color-dark"),
          backgroundColor: style.getPropertyValue("--primary-color"),
        },
      ],
    },
    options: {
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
          beginAtZero: true,
          ticks: {
            callback: function(value) {
              if (value % 1 !== 0) return;
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
