
Mini Hugo:
- We have a bunch of markdown files in the folder
- We have a landing page (index.md) which will be turned into index.html
- We transpile each markdown file into html
- We have (go) html templates (so 1 template for index.html, 1 template for every other page)
- Github pages deployment (for now anyways)
- What do we do with assets? (styles, javascript, images, etc)

- blog init     --> initialize folder structure
- blog build    --> build the html files
    - --preview -> open the browser window to view the site
    - --publish -> push to github

blog/
|  src/    (holds all the .md files)
|  site/   (holds all the html files)
|  assets/ (holds all the assets)
| .github/ (github pages stuff)