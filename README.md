# Simple Jekyll Personal Site

A minimal Jekyll site for a personal homepage and a list of articles.

## Customize

1. Edit `_config.yml`.
2. Edit `index.md` to add your name and introduction.
3. Replace `assets/img/profile.jpg` with your profile picture.
4. Edit the social links in `_layouts/post.html`.
5. Add articles to `_posts/` using the format:

   `YYYY-MM-DD-title.md`

Each post needs YAML front matter, for example:

```yaml
---
title: "My Article"
date: 2026-09-20
---
```

## Run locally

Install Ruby and Bundler, then:

```bash
bundle install
bundle exec jekyll serve
```

Open http://localhost:4000.

## GitHub Pages

Push the repository to GitHub with `main` as the default branch. In the
repository settings, open Pages and select **GitHub Actions** as the source.

The included workflow builds and deploys the site on every push to `main`.
