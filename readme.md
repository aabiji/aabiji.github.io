This repository contains the source code for my blog. 

Project structure:
- ./.github/workflows -- Contains the custom github action used to build the website.
- ./static            -- Contains some website templates and the blog's data.
- ./web               -- Contains all the html files. This is the website's root folder.
- ./src               -- Source code for my custom static site generator.
- ./posts             -- Contains the markdown sources for my blog posts.

Build instructions:
```fish
# Install
cd /path/to/project
cargo build --release
mv target/release/blog .

# Now run
blog --help
```
