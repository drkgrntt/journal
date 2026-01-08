(function() {
  const journals = JSON.parse(document.getElementById("mood-vs-thankfulness-journals").textContent);
  const element = document.getElementById("mood-vs-thankfulness");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const mapping = journals.reduce(function(acc, journal) {
    const date = new Date(journal.date).toLocaleDateString();
    acc[date] ||= {
      thankful: false,
      rating: []
    };
    acc[date].rating.push(journal.rating.value)
    if ((journal.thankfuls?.length ?? 0) > 0) {
      acc[date].thankful = true
    }
    return acc
  }, {})

  const thankfulData = [1, 2, 3, 4, 5].map(function(rating) {
    return Object.values(mapping).reduce(function(acc, { thankful, rating: journalRatings }) {
      const avgRating = journalRatings.reduce(function(acc, rating) {
        return acc + rating;
      }, 0) / journalRatings.length

      if (Math.round(avgRating) === rating) {
        acc += thankful ? 1 : 0
      }

      return acc;
    }, 0)
  })

  const notThankfulData = [1, 2, 3, 4, 5].map(function(rating) {
    return Object.values(mapping).reduce(function(acc, { thankful, rating: journalRatings }) {
      const avgRating = journalRatings.reduce(function(acc, rating) {
        return acc + rating;
      }, 0) / journalRatings.length

      if (Math.round(avgRating) === rating) {
        acc += thankful ? 0 : 1
      }

      return acc;
    }, 0)
  })

  const chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: ["Awful", "Bad", "Fine", "Good", "Great"],
      datasets: [
        {
          label: "Thankful Items",
          data: thankfulData,
          borderColor: getCssValue("--primary-color-dark"),
          backgroundColor: getCssValue("--primary-color"),
        },
        {
          label: "No Thankful Items",
          data: notThankfulData,
          borderColor: getCssValue("--secondary-color-dark"),
          backgroundColor: getCssValue("--secondary-color"),
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      minBarLength: 2,
      plugins: {
        title: {
          display: true,
          text: "Average Rating vs Thankfulness",
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
        },
      },
    },
  })

  // Show the chart
  element.innerHTML = "";
  element.appendChild(canvas);
})();
