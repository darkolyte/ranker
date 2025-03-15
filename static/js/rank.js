window.addEventListener('DOMContentLoaded', (e) => {

    if (window.location.href.includes("/collections/") !== true) return

    let collection = document.getElementsByClassName("abc")

    const pairs = []

    for (i = 0; i < collection.length; i++) {
        for (j = i + 1; j < collection.length; j++) {
            pairs.push([collection[i].id, collection[j].id])
        }
    }

    // maybe fetch these by id
    let left = document.getElementsByClassName("left")[0]
    let right = document.getElementsByClassName("right")[0]

    let scores = {}

    function processNextPair() {
        if (pairs.length === 0) {
            // maybe fetch these by id
            const form = document.getElementsByTagName("form")[0]
            const input = document.getElementsByTagName("input")[0]
            input.value = JSON.stringify(scores);
            form.submit()
            return;
        }

        const pair = pairs.shift();

        left.innerHTML = document.getElementById(pair[0]).innerHTML;
        left.setAttribute("data-id", pair[0])
        right.innerHTML = document.getElementById(pair[1]).innerHTML;
        right.setAttribute("data-id", pair[1])

        const handleClick = (event) => {
            left.removeEventListener('click', handleClick);
            right.removeEventListener('click', handleClick);

            const leftId = left.getAttribute("data-id")
            const rightId = right.getAttribute("data-id")

            left.style.transition = "all 500ms linear"
            right.style.transition = "all 500ms linear"
            
            let chosenId
            

            if (event.target === left) {
                chosenId = pair[0]
                left.style.backgroundColor = "green"
               
              
            } else {
                chosenId = pair[1]
                right.style.backgroundColor = "green"               
            }

            scores[chosenId] = (scores[chosenId] || 0) + 1;           

            setTimeout(() => {

                
                left.style.transition = "all 500ms cubic-bezier(.16,.8,.19,.98)"
                right.style.transition = "all 500ms cubic-bezier(.16,.8,.19,.98)"

                left.style.transform = "translateX(250%)"
                right.style.transform = "translateX(250%)"

                processNextPair();

               setTimeout(() => {

                left.style.transition = "none"
                right.style.transition = "none"

                left.style.backgroundColor = "white"
                right.style.backgroundColor = "white"

                left.style.transform = "translateX(-250%)"
                right.style.transform = "translateX(-250%)"

                setTimeout(() => {
                    left.style.transition = "all 500ms cubic-bezier(.48,.52,.24,.9)"
                    right.style.transition = "all 500ms cubic-bezier(.48,.52,.24,.9)"
                
                    left.style.transform = "translateX(0%)"
                    right.style.transform = "translateX(0%)"

                }, 100)
           
                }, 500);
                
            }, 500)
            
        };

        left.addEventListener('click', handleClick);
        right.addEventListener('click', handleClick);
    }

    processNextPair();
})