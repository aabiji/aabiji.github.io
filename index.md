---
layout: default
title: Home
---

<section class="intro">
  <div class="intro-text">
    <h1>Your Name</h1>

    <p>
      I'm a writer, researcher, developer, or whatever description
      makes sense for you. This is a short introduction to who you are
      and what you write about.
    </p>
  </div>

  <img
    class="profile"
    src="{{ '/assets/img/profile.jpg' | relative_url }}"
    alt="Your Name"
  >
</section>

<section class="articles">
  <h2>Articles</h2>

  <ul class="article-list">
    {% for post in site.posts %}
      <li>
        <time datetime="{{ post.date | date_to_xmlschema }}">
          {{ post.date | date: "%Y-%m-%d" }}
        </time>

        <a href="{{ post.url | relative_url }}">{{ post.title }}</a>
      </li>
    {% endfor %}
  </ul>
</section>
