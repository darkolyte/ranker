window.addEventListener('DOMContentLoaded', (e) => { 
    let addCol = document.getElementById("addCol")
    let modal = document.getElementById("modal")
    let titleInput = document.getElementById("titleInput")

    addCol.addEventListener("click", (e) => {
        modal.style.display = "block";
        titleInput.focus()
    })

    window.onclick = function(event) {
        if (event.target == modal) {
          modal.style.display = "none";
        }
      }
})