(function() {
  const data = JSON.parse(document.getElementById("mood-chart-data").textContent);
  const element = document.getElementById("mood-chart");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "line",
    data: {
      labels: data.map(item => new Date(item.date).toLocaleDateString()),
      datasets: [
        {
          label: "Rating Flow",
          data: data.map(item => item.value),
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
