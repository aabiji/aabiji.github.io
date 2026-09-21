---
layout: default
title: Home
---
<section class="home-intro">
  <div class="home-copy">
    <h1>Abigail Adegbiji</h1>
    <p>I'm an engineering student at USask. I'll occasionally write things I find interesting here.</p>
    <nav class="socials" aria-label="Social links">
      <a href="https://github.com/aabiji" aria-label="GitHub"><i class="fa-brands fa-github"></i></a>
      <a href="https://x.com/TripleAAABaddie" aria-label="X"><i class="fa-brands fa-square-x-twitter"></i></a>
      <a href="https://www.linkedin.com/in/abigail-adegbiji-96514540a/" aria-label="LinkedIn"><i class="fa-brands fa-linkedin"></i></a>
      <button class="theme-toggle" type="button" aria-label="Toggle theme"><i class="fa-solid fa-circle-half-stroke"></i></button>
    </nav>
  </div>
  <img class="profile" src="{{ '/assets/img/profile.jpg' | relative_url }}" alt="Profile pic">
</section>
<section class="articles">
  <h2>Articles</h2>
  <ul class="article-list">
    {% for post in site.posts %}
    <li><a href="{{ post.url | relative_url }}">{{ post.title }}</a><time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %Y" }}</time></li>
    {% endfor %}
  </ul>
</section>
