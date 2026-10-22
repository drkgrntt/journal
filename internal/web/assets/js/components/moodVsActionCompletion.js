(function() {
  const data = JSON.parse(document.getElementById("mood-vs-action-completion-data").textContent);
  const element = document.getElementById("mood-vs-action-completion");

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
          label: "Completed Action",
          data: data.map(item => item.withCompletions),
          ...themedColors("--primary-color-dark", "--primary-color-light"),
          borderWidth: 2,
        },
        {
          label: "No Completed Action",
          data: data.map(item => item.withoutCompletions),
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
        },
      },
    },
  })

  // Show the chart
  element.innerHTML = "";
  element.appendChild(canvas);
})();
