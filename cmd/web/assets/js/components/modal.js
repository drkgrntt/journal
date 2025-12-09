function getModals() {
  return Array.from(document.querySelectorAll("[modal]"))
}

function loadModals() {
  getModals().forEach(function(modal) {
    const buttonId = `${modal.id}-button`
    const button = document.getElementById(buttonId)

    button.onclick = function() {
      modal.showModal()
    }

    const close = modal.querySelector("[close]")
    close.onclick = function() {
      modal.close()
    }

    modal.addEventListener("close", function() {
      modal.close()
    })

    modal.addEventListener("click", function(event) {
      event.stopPropagation()
      const dialogRect = modal.getBoundingClientRect();
      if (
        event.clientX < dialogRect.left ||
        event.clientX > dialogRect.right ||
        event.clientY < dialogRect.top ||
        event.clientY > dialogRect.bottom
      ) {
        modal.close();
      }
    });
  })
}

htmx.onLoad(function(e) {
  loadModals()
})

loadModals()
