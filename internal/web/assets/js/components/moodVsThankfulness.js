(function() {
  const data = JSON.parse(document.getElementById("mood-vs-thankfulness-data").textContent);
  const element = document.getElementById("mood-vs-thankfulness");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: data.map(item => item.rating),
      datasets: [
        {
          label: "Thankful Items",
          data: data.map(item => item.withShare),
          ...themedColors("--primary-color-dark", "--primary-color-light"),
          borderWidth: 2,
        },
        {
          label: "No Thankful Items",
          data: data.map(item => item.withoutShare),
          ...themedColors("--secondary-color-dark", "--secondary-color-light"),
          borderWidth: 2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      minBarLength: 2,
      plugins: {
        tooltip: {
          callbacks: {
            label: function(context) {
              const item = data[context.dataIndex];
              const isWith = context.datasetIndex === 0;
              const count = isWith ? item.withThankfuls : item.withoutThankfuls;
              const share = Math.round(context.parsed.y);
              return `${share}% of days ${isWith ? "with a thankful" : "without a thankful"} (${count} ${count === 1 ? "day" : "days"})`;
            },
          },
        },
      },
      scales: {
        x: {
          grid: {
            display: false,
          },
        },
        y: {
          min: 0,
          max: 100,
          grid: {
            display: false,
          },
          ticks: {
            callback: value => `${value}%`,
          },
        },
      },
    },
  })

  // Show the chart
  element.innerHTML = "";
  element.appendChild(canvas);
})();
