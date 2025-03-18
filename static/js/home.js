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
    let collections = document.getElementsByClassName("col")

    for (i = 0; i < detailPanes.length; i ++) {
        if (cId === detailPanes[i].dataset.id) {
            selectedCollection = detailPanes[i]
            collections[i].style.border = "4px solid dimgray"
        } else {
            detailPanes[i].style.display = "none"
        }
    }

    if (selectedCollection === null) {
        selectedCollection = detailPanes[0]
        selectedCollection.style.display = "flex"
        collections[0].style.border = "4px solid dimgray"
    }

    for (i = 0; i < collections.length; i ++) {
        collections[i].addEventListener('click', (e) => {
            e.target.style.border = "4px solid dimgray"
            for (j = 0; j < detailPanes.length; j++) {
                if (detailPanes[j].dataset.id === e.target.dataset.id) {
                    selectedCollection.style.display = "none"
                    selectedCollection = detailPanes[j]
                    // window.scroll({
                    //     top: selectedCollection.getBoundingClientRect().top + window.scrollY,
                    //     behavior: "smooth"
                    // })
                } else {
                    collections[j].style.border = "4px solid transparent"
                }
            }
            selectedCollection.style.display = "flex"
            updateURL(parseInt(selectedCollection.dataset.id))
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