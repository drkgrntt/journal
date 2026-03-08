function initMoodByTopic() {
  const data = JSON.parse(document.getElementById("mood-by-topic-data").textContent);
  const element = document.querySelector(".mood-by-topic");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: data.map(item => item.topic),
      datasets: [
        {
          label: "Average Rating",
          data: data.map(item => item.value),
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
        title: {
          display: true,
          text: "Average Rating By Topic",
          position: "bottom",
        },
      },
      scales: {
        x: {
          grid: {
            display: false,
          },
        },
        y: {
          min: Math.floor(Math.min(...data.map(item => item.value))) - .3,
          max: Math.ceil(Math.max(...data.map(item => item.value))) + .3,
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
}
document.addEventListener("load-mood-by-topic", initMoodByTopic)
