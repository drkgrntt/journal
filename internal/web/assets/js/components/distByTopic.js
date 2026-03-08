function initDistByTopic() {
  const data = JSON.parse(document.getElementById("dist-by-topic-data").textContent);
  const element = document.querySelector(".dist-by-topic");

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
          label: "Quantity",
          data: data.map(item => item.quantity),
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
          text: "Distribution By Topic",
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
document.addEventListener("load-dist-by-topic", initDistByTopic)
