htmx.onLoad(function(_e) {
	const toConvert = document.querySelectorAll("[data-date], [data-datetime], [data-time]")

	toConvert.forEach(function(element) {
		switch (true) {
			case !!element.dataset.date:
				element.innerHTML = new Date(element.dataset.date).toLocaleDateString(undefined, {
					year: "numeric",
					month: "long",
					day: "numeric"
				})
				break;
			case !!element.dataset.datetime:
				element.innerHTML = new Date(element.dataset.datetime).toLocaleString(undefined, {
					dateStyle: "medium",
					timeStyle: "short"
				})
				break;
			case !!element.dataset.time:
				element.innerHTML = new Date(element.dataset.time).toLocaleTimeString(undefined, {
					hour: "numeric",
					minute: "numeric"
				})
				break;
		}
	})
})
