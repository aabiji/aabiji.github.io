This repository contains the source code for my blog. 

Project structure:
- ./.github/workflows -> Contains the custom github action used to build the website.
- ./static            -> Contains some website templates and the blog's data.
- ./web               -> Contains all the html files. This is the website's root folder.
- ./src               -> Source code for my custom static site generator.
- ./posts             -> Contains the markdown sources for my blog posts.

Build instructions:
```fish
# Install
cargo build --release
set -x PATH $PATH:/home/user/path/to/project_folder/target/release

# Now run
blog --help
```

Todo:
- Rss
- Better into
- Hello world post
- Github pages deployment
- autoamtically create files if they don't already exist
- improve the rust code
- improve the direcoty structure
- rewrite this post
    - Explain how this blog was built
    - Explain why I'm writing
