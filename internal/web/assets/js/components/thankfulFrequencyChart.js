function initThankfulFrequencyChart() {
  const data = JSON.parse(document.getElementById("thankful-frequency-chart-data").textContent);
  const element = document.querySelector(".thankful-frequency-chart");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "line",
    data: {
      labels: data.map(item => item.date),
      datasets: [
        {
          label: "Thankfuls",
          data: data.map(item => item.quantity),
          borderColor: getCssValue("--primary-color-dark"),
          backgroundColor: getCssValue("--primary-color"),
        },
      ],
    },
    options: {
      spanGaps: true,
      tension: .4,
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: false
        }
      },
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
        }
      },
    },
  })

  // Show the chart
  element.innerHTML = "";
  element.appendChild(canvas);
}
document.addEventListener("load-thankful-frequency-chart", initThankfulFrequencyChart)
