window.addEventListener('DOMContentLoaded', (e) => {

    let collection = document.getElementsByClassName("items")
    let left = document.getElementById("left")
    let right = document.getElementById("right")

    if (collection.length === 0) {
        left.style.display = "none"
        right.style.display = "none"

        let el = document.getElementsByClassName("choice-holder")[0]

        el.innerHTML = "No items in this collection :("
        return
    }

    const pairs = []

    for (i = 0; i < collection.length; i++) {
        for (j = i + 1; j < collection.length; j++) {
            pairs.push([collection[i].id, collection[j].id])
        }
    }

    let scores = {}

    function processNextPair() {
        if (pairs.length === 0) {
            const form = document.getElementById("result")
            const input = document.getElementById("scores")
            input.value = JSON.stringify(scores);
            form.submit()
            return;
        }

        const pair = pairs.shift();

        left.innerHTML = document.getElementById(pair[0]).innerHTML;
        right.innerHTML = document.getElementById(pair[1]).innerHTML;

        const handleClick = (event) => {
            left.removeEventListener('click', handleClick);
            right.removeEventListener('click', handleClick);

            left.style.transition = "all 500ms linear"
            right.style.transition = "all 500ms linear"
            
            let chosenId = event.target === left ? pair[0]: pair[1];
            event.target.style.backgroundColor = "green";
            scores[chosenId] = (scores[chosenId] || 0) + 1;           

            setTimeout(() => {
                
                left.style.transition = "all 500ms cubic-bezier(.16,.8,.19,.98)"
                right.style.transition = "all 500ms cubic-bezier(.16,.8,.19,.98)"

                left.style.transform = "translateX(250%)"
                right.style.transform = "translateX(250%)"

               setTimeout(() => {

                left.style.transition = "none"
                right.style.transition = "none"

                left.style.backgroundColor = ""
                right.style.backgroundColor = ""

                left.style.transform = "translateX(-250%)"
                right.style.transform = "translateX(-250%)"

                left.innerHTML = ""
                right.innerHTML = ""

                setTimeout(() => {
                    left.style.transition = "all 500ms cubic-bezier(.48,.52,.24,.9)"
                    right.style.transition = "all 500ms cubic-bezier(.48,.52,.24,.9)"
                
                    left.style.transform = "translateX(0%)"
                    right.style.transform = "translateX(0%)"

                    processNextPair();
                }, 100)
           
                }, 500);
                
            }, 500)
            
        };

        left.addEventListener('click', handleClick);
        right.addEventListener('click', handleClick);
    }

    processNextPair();
})