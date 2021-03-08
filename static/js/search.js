/*!
 * This is used to filter out/shorten the huge cloud of keywords as the user
 * types into the input box. It's a cheap search function.
 * This is only doing a substring match on keyword name. In the near future, it
 * should be able to show keywords where the substring was in the link titles.
 */
const searchBar = document.getElementById("go2input");
const keywordsList = document.getElementById("keywordslist");

let inputKeywords = [];  // strings the user is entering in the main input box
var keywordsArray = [];  // list of lists. [ [keywordObject, len(lists), Clicks], ...]

searchBar.addEventListener('input', (e) => {
    const searchString = e.target.value.toLowerCase();

    const filteredKeywords = inputKeywords.filter((keyword) => {
        return (
            keyword[0].toLowerCase().includes(searchString)
        );
    });
    displayKeywords(filteredKeywords);
});

const loadKeywords = async () => {
  
    try {
        const res = await fetch('/api/keywords');
        keywords = await res.json();
        // try to get to membership list length per keyword
        for (const [key, value] of Object.entries(keywords)) {
          // item[0] is going to be the keyword string
          // item[1] is going to be the length of keyword.Links (number of links in the list)
          // item[2] is going to be click count on the keyword
          keywordsArray.push([key, Object.keys(value["Links"]).length, value.Clicks]);
        }
        inputKeywords = keywordsArray;
        displayKeywords(inputKeywords);
    } catch (err) {
        console.error(err);
    }
};

const displayKeywords = (keywords) => {

    // sort by the third element of each array (click count)
    keywords.sort(function(a, b) {
        var valueA, valueB;
        valueA = a[2]; 
        valueB = b[2];
        if (valueA < valueB) {
            return -1;
        }
        else if (valueA > valueB) {
            return 1;
        }
        return 0;
    });

    keywords.reverse(); // descending

    const htmlString = keywords
        .map((keyword) => {
          if (keyword[1] > 1) {
            return `
            <li class="list-inline-item">
                <a class="go2keyword go2keyword-small" href="/.${keyword[0]}" title="${keyword[1]} links, ${keyword[2]} clicks" role="button">go2/${keyword[0]}</a><sup>${keyword[1]}</sup>
            </li>
            `;
          }
          else {
            return `
            <li class="list-inline-item">
                <a class="go2keyword go2keyword-small" href="/.${keyword[0]}" title="1 link, ${keyword[2]} clicks" role="button">go2/${keyword[0]}
            </li>
            `;
          }
        })
        .join('');
    keywordsList.innerHTML = htmlString;
};

loadKeywords();
