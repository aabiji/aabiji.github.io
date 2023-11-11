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
    html: String,
    markdown: String,
    read_time: String,
    publish_date: String,
}

impl Post {
    fn new(title: &str, path: &str, date: &str) -> Self {
        Post {
            path: String::from(path),
            title: String::from(title),
            publish_date: String::from(date),
            html: String::new(),
            markdown: String::new(),
            read_time: String::new(),
        }
    }

    fn init(path: &str) -> Self {
        let markdown = std::fs::read_to_string(&path).unwrap();
        let html = transpile_markdown(&markdown);

        let path_parts: Vec<&str> = path.split(".").collect();
        let path = path_parts[0].to_owned() + ".html";

        let time = chrono::Utc::now();
        let publish_date = time.format("%B %d, %Y").to_string();

        let title_line = markdown.lines().next().unwrap();
        let title = title_line[2..].to_string();

        let wpm = 200.0; // Number of words per minute the average person reads
        let words = (get_word_count(&markdown) as f64) / wpm;
        let read_time = format!("{} min", words as i32);

        Post { path, title, publish_date, read_time, markdown, html }
    }
}

#[derive(Serialize)]
struct Blog {
    json: Value,
    posts: Vec<Post>,
    json_path: String,
    removed_post: String,
}

impl Blog {
    fn new(json_path: &str) -> Self {
        Blog {
            json: Value::Null,
            posts: Vec::new(),
            removed_post: String::new(),
            json_path: String::from(json_path)
        }
    }

    fn load(&mut self) {
        let json_str = std::fs::read_to_string(&self.json_path).unwrap();
        self.json = serde_json::from_str(&json_str).unwrap();

        for (key, value) in self.json["posts"].as_object().unwrap() {
            if key == &self.removed_post {
                continue;
            }

            let p = Post::new(key, &value["path"].to_string(), &value["publish_date"].to_string());
            self.posts.push(p);
        }
    }

    fn build_post(&mut self, path: &str) {
        let mut tera = Tera::default();
        tera.add_template_file("templates/post.template", Some("Post")).unwrap();

        let p = Post::init(path);
        let context = Context::from_serialize(&p).unwrap();
        let html = tera.render("Post", &context).unwrap();
        std::fs::write(&p.path, html).unwrap();

        self.posts.push(p);
    }

    fn build(&mut self) {
        let mut tera = Tera::default();
        tera.add_template_file("templates/index.template", Some("Index")).unwrap();

        let mut context = Context::new();
        context.insert("posts", &self.posts);
        let html = tera.render("Index", &context).unwrap();
        std::fs::write("index.html", html).unwrap();
    }
}

fn print_help() {
    println!("blog info goes here");
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() == 1 {
        print_help();
        return;
    }

    let mut blog = Blog::new("archive.json");

    if &args[1] == "publish" {
        blog.build_post(&args[2]);
    } else if &args[1] == "remove" {
        blog.removed_post = String::from(&args[2]);
    } else {
        print_help();
        return;
    }

    blog.load();
    blog.build();
}
