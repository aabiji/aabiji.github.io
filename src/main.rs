use std::collections::HashMap;

use tera::Tera;
use tera::Context;

use comrak::nodes::{AstNode, NodeValue};

use serde_json::Value;
use serde::Serialize;

// Traverse the AST counting the number of words in each text node
fn count_words<'a>(node: &'a AstNode<'a>) -> usize {
    let mut word_count = match &node.data.borrow().value {
        NodeValue::Text(text) => {
            let words: Vec<&str> = text.split(" ").collect();
            words.len()
        },
        _ => {0},
    };

    for child in node.children() {
        word_count += count_words(child);
    }

    word_count
}

// Get the number of words in a markdown file
fn get_word_count(markdown: &str) -> usize {
    let arena = comrak::Arena::new();
    let options = comrak::Options::default();
    let root = comrak::parse_document(&arena, &markdown, &options);

    count_words(root)
}

// Transpile markdown into html
fn transpile_markdown(file_contents: &str) -> String {
    let arena = comrak::Arena::new();
    let options = comrak::Options::default();
    let root = comrak::parse_document(&arena, &file_contents, &options);

    let mut html_bytes = vec![];
    comrak::format_html(root, &options, &mut html_bytes).unwrap();
    String::from_utf8(html_bytes).unwrap()
}

#[derive(Serialize)]
struct Post {
    path: String,
    title: String,
    read_time: String,
    publish_date: String,
    html: String,
    markdown: String
}

impl Post {
    fn new() -> Self {
        Post {
            path: String::new(),  
            title: String::new(),  
            read_time: String::new(),  
            publish_date: String::new(),  
            html: String::new(),  
            markdown: String::new(),  
        }
    }

    fn init(path: &str) -> Self {
        let markdown = std::fs::read_to_string(&path).unwrap();
        let html = transpile_markdown(&markdown);

        let time = chrono::Utc::now();
        let publish_date = time.format("%B %d, %Y").to_string();

        let title_line = markdown.lines().next().unwrap();
        let title = title_line[2..].to_string();

        let path = "web/".to_owned() + &title.replace(" ", "_") + ".html";

        let wpm = 200.0; // Number of words the average person reads
        let words = (get_word_count(&markdown) as f64) / wpm;
        let read_time = format!("{} min", words as i32);

        Post { path, title, publish_date, read_time, markdown, html }
    }
}

#[derive(Serialize)]
struct Blog {
    posts: HashMap<String, Post>,
    posts_cache_path: String
}

impl Blog {
    fn new(posts_cache_path: &str) -> Self {
        let mut blog = Blog {
            posts: HashMap::new(),
            posts_cache_path: String::from(posts_cache_path)
        };

        blog.load_archive();
        blog
    }

    fn load_archive(&mut self) {
        let json_str = std::fs::read_to_string(&self.posts_cache_path).unwrap();
        if json_str.len() == 0 {
            return;
        }
        let json: Value = serde_json::from_str(&json_str).unwrap();

        for (key, value) in json.as_object().unwrap() {
            let mut p = Post::new();

            p.title = key.to_string();
            p.path = value["path"].to_string();
            p.publish_date = value["publish_date"].to_string();
            self.posts.insert(key.to_string(), p);
        }
    }

    fn save_archive(&self) {
        let json = serde_json::to_string(&self.posts).unwrap();
        std::fs::write(&self.posts_cache_path, json).unwrap();
    }

    fn remove_post(&mut self, post_title: &str) {
        self.posts.remove(post_title);
    }

    fn add_post(&mut self, path: &str) {
        let mut tera = Tera::default();
        let template = "static/post.template";
        tera.add_template_file(template, Some("Post")).unwrap();

        let mut p = Post::init(path);
        let context = Context::from_serialize(&p).unwrap();

        let html = tera.render("Post", &context).unwrap();
        std::fs::write(&p.path, html).unwrap();

        p.html = String::new();
        p.markdown = String::new();
        self.posts.insert(p.title.clone(), p);
    }

    fn build(&mut self) {
        let mut tera = Tera::default();
        let template = "static/index.template";
        tera.add_template_file(template, Some("Index")).unwrap();

        let mut context = Context::new();
        context.insert("posts", &self.posts);

        let html = tera.render("Index", &context).unwrap();
        std::fs::write("index.html", html).unwrap();
    }
}

fn print_help() {
    println!("{}", "
blog
-----
A super simple static site geneator for my blog.

To create or update a post:
'blog publish example_file.md'
The first line in the file should be a header containing the post's title. Ex.
'
# Post title
 <The post's content goes here>
'

To remove a post:
'blog remove 'Post title''
    ");
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() == 1 {
        print_help();
        return;
    }

    let mut blog = Blog::new("static/posts.json");

    if &args[1] == "publish" {
        blog.add_post(&args[2]);
    } else if &args[1] == "remove" {
        blog.remove_post(&args[2]);
    } else {
        print_help();
        return;
    }

    blog.build();
    blog.save_archive();
}
