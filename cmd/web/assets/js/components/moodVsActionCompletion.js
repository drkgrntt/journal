(function() {
  const journals = JSON.parse(document.getElementById("mood-vs-action-completion-journals").textContent);
  const actionItems = JSON.parse(document.getElementById("mood-vs-action-completion-action-items").textContent);
  const element = document.getElementById("mood-vs-action-completion");

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Canvas not supported");
  }

  const mapping = journals.reduce(function(acc, journal) {
    const date = new Date(journal.date).toLocaleDateString();
    acc[date] ||= {
      completedActionItem: false,
      rating: []
    };
    acc[date].rating.push(journal.rating.value)
    return acc
  }, {})

  actionItems.forEach(function(actionItem) {
    if (!actionItem.completedAt) return;
    const date = new Date(actionItem.completedAt).toLocaleDateString();
    mapping[date] ||= {
      completedActionItem: false,
      rating: []
    }
    mapping[date].completedActionItem = true;
  })

  const completedData = [1, 2, 3, 4, 5].map(function(rating) {
    return Object.values(mapping).reduce(function(acc, { completedActionItem, rating: journalRatings }) {
      const avgRating = journalRatings.reduce(function(acc, rating) {
        return acc + rating;
      }, 0) / journalRatings.length

      if (Math.round(avgRating) === rating) {
        acc += completedActionItem ? 1 : 0
      }

      return acc;
    }, 0)
  })

  const notCompletedData = [1, 2, 3, 4, 5].map(function(rating) {
    return Object.values(mapping).reduce(function(acc, { completedActionItem, rating: journalRatings }) {
      const avgRating = journalRatings.reduce(function(acc, rating) {
        return acc + rating;
      }, 0) / journalRatings.length

      if (Math.round(avgRating) === rating) {
        acc += completedActionItem ? 0 : 1
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
          label: "Completed Action",
          data: completedData,
          borderColor: getCssValue("--primary-color-dark"),
          backgroundColor: getCssValue("--primary-color-light"),
          borderWidth: 2,
        },
        {
          label: "No Completed Action",
          data: notCompletedData,
          borderColor: getCssValue("--secondary-color-dark"),
          backgroundColor: getCssValue("--secondary-color-light"),
          borderWidth: 2,
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
          text: "Average Rating vs Action Completion",
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
