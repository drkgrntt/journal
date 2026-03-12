function initRoutineCompletionRate() {
  const data = JSON.parse(document.getElementById("routine-completion-rate-data").textContent);
  const element = document.querySelector(".routine-completion-rate");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: data.map(item => item.routine),
      datasets: [
        {
          label: "Completion Rate",
          data: data.map(item => item.percent.toFixed(2)),
          borderColor: getCssValue("--secondary-color-dark"),
          backgroundColor: getCssValue("--secondary-color-light"),
          borderWidth: 2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: false,
        },
      },
      scales: {
        x: {
          grid: { 
            display: false,
          },
          ticks: {
            callback: function(value, index) {
              const shouldBlur = localStorage.getItem(`blur-text-${document.body.id}`) === "true";
              return shouldBlur ? '' : this.getLabelForValue(value);
            }
          }
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
document.addEventListener("load-routine-completion-rate", initRoutineCompletionRate)
