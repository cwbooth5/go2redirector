/*!
 * This is used to filter out/shorten the huge cloud of keywords as the user
 * types into the input box. It's a cheap search function.
 * This is only doing a substring match on keyword name. In the near future, it
 * should be able to show keywords where the substring was in the link titles.
 */
const searchBar = document.getElementById("go2input");
const keywordsList = document.getElementById("keywordslist");



let inputKeywords = [];  // strings the user is entering in the main input box
var keywordsArray = [];  // list of lists. [ [keywordObject, len(lists)], ...]

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
          keywordsArray.push([key, Object.keys(value["Links"]).length])
        }
        inputKeywords = keywordsArray;
        displayKeywords(inputKeywords);
    } catch (err) {
        console.error(err);
    }
};

const displayKeywords = (keywords) => {

    const htmlString = keywords
        .map((keyword) => {
            return `
            <li class="list-inline-item">
                <a class="go2keyword go2keyword-small" href="/.${keyword[0]}" title="TODO clicks, TODO links" role="button">go2/${keyword[0]}</a><sup>${keyword[1]}</sup>
            </li>
        `;
        })
        .join('');
    keywordsList.innerHTML = htmlString;
};

loadKeywords();
