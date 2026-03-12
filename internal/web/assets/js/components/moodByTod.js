function initMoodByTod() {
  const data = JSON.parse(document.getElementById("mood-by-tod-data").textContent);
  const element = document.querySelector(".mood-by-tod");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  function convertTo12Hour(time) {
    const [hours, minutes] = time.split(":");
    const hoursInt =parseInt(hours) 
    const period = hoursInt >= 12 ? "PM" : "AM";
    const hours12 = hoursInt % 12 || 12;
    return `${hours12}:${minutes} ${period}`;
  }

  const chart = new Chart(ctx, {
    type: "line",
    data: {
      labels: data.map(item => convertTo12Hour(item.tod)),
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
      tension: 0.4,
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
document.addEventListener("load-mood-by-tod", initMoodByTod)
