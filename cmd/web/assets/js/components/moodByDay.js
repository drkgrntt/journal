(function() {
  const journals = JSON.parse(document.getElementById("mood-by-day-journals").textContent);
  const element = document.getElementById("mood-by-day");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const days = [
    "Sunday",
    "Monday",
    "Tuesday",
    "Wednesday",
    "Thursday",
    "Friday",
    "Saturday",
  ]
  const mapping = {}
  journals.forEach(function(journal) {
    const key = days[new Date(journal.date).getDay()];
    mapping[key] ||= [];
    mapping[key].push({
      rating: journal.rating.value,
    });
  });

  const ratingData = days.map(function(day) {
    const rate = mapping[day].reduce(function(acc, { rating }) {
      return acc + rating;
    }, 0);
    return rate / mapping[day].length;
  });

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: days,
      datasets: [
        {
          label: "Average By Day",
          data: ratingData,
          borderColor: getCssValue("--secondary-color-dark"),
          backgroundColor: getCssValue("--secondary-color"),
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
