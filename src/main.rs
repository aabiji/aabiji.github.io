use tera::Tera;
use tera::Context;
use serde::Serialize;
use comrak::nodes::{AstNode, NodeValue};

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
    fn new(path: &str) -> Self {
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

fn build() {
    let post = Post::new("test.md");

    let file = "post.template";
    let mut tera = Tera::default();
    tera.add_template_file("templates/".to_owned() + file, Some(&file)).unwrap();

    let final_html = tera.render(&file, &Context::from_serialize(&post).unwrap()).unwrap();
    std::fs::write(post.path, final_html).unwrap();
}

fn main() {
    build();
}
