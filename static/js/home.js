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

    let cId = document.location.pathname.slice(1);
    let selectedCollection = null

    let detailPanes = document.getElementsByClassName("col-details")

    for (i = 0; i < detailPanes.length; i ++) {
        if (cId === detailPanes[i].dataset.id) {
            selectedCollection = detailPanes[i]
        } else {
            detailPanes[i].style.display = "none"
        }
    }

    if (selectedCollection === null) {
        selectedCollection = detailPanes[0]
        selectedCollection.style.display = "flex"

    }

    let collections = document.getElementsByClassName("col")

    for (i = 0; i < collections.length; i ++) {
        collections[i].addEventListener('click', (e) => {

            for (i = 0; i < detailPanes.length; i ++) {
                if (detailPanes[i].dataset.id === e.target.dataset.id) {
                    selectedCollection.style.display = "none"
                    detailPanes[i].style.display = "flex"
                    selectedCollection = detailPanes[i]
                    updateURL(parseInt(selectedCollection.dataset.id))
                }
            }
        })
    }
    
    function updateURL(newId) {
        let urlParts =  window.location.pathname.split("/")

        while (urlParts.length != 2) {
            urlParts.pop()
        }

        urlParts[1] = newId

        const newUrl = urlParts.join("/")

        // window.history.replaceState(null, "", newUrl)
        window.history.pushState(null, "", newUrl)
    }

})