use std::collections::HashMap;
use std::io::BufReader;
use std::path::Path;

use tera::Tera;
use tera::Context;

use comrak::nodes::{AstNode, NodeValue};

use serde_json::Value;
use serde::Serialize;

use rss::{Channel, Item};

const HTML_DIRECTORY: &str  = "web/";
const RSS_PATH: &str        = "web/rss.xml";

const POST_CACHE_PATH: &str = "static/posts.json";
const HOME_TEMPLATE: &str   = "static/index.template";
const POST_TEMPLATE: &str   = "static/post.template";

const BLOG_URL: &str        = "https://aabiji.github.io/";
const BLOG_TITLE: &str      = "Some thoughts";

fn check_paths() {
    if !Path::new(HTML_DIRECTORY).exists() {
        std::fs::create_dir(HTML_DIRECTORY).unwrap();
    }

    if !Path::new(HOME_TEMPLATE).exists() || !Path::new(POST_TEMPLATE).exists() {
        let msg = "Template files (index.template and post.template) in static/ not found.";
        panic!("{}", msg);
    }

    if !Path::new(POST_CACHE_PATH).exists() {
        let _ = std::fs::File::create(POST_CACHE_PATH);
    }

    if !Path::new(RSS_PATH).exists() {
        let _ = std::fs::File::create(RSS_PATH);
    }
}

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

        let path = title.replace(" ", "_") + ".html";

        let wpm = 100.0; // Number of words the average person reads
        let words = (get_word_count(&markdown) as f64) / wpm;
        let read_time = format!("{} min", words as i32);

        Post { path, title, publish_date, read_time, markdown, html }
    }
}

struct RSSFeed {
    channel: rss::Channel
}

impl RSSFeed {
    fn new() -> Self {
        let file = std::fs::File::open(RSS_PATH).unwrap();
        let channel = match Channel::read_from(BufReader::new(file)) {
            Ok(c) => c,
            Err(_) => {
                // Create a new channel on EOF error
                let mut new_channel = Channel::default();
                new_channel.set_link(BLOG_URL);
                new_channel.set_title(BLOG_TITLE);
                new_channel
            }
        };

        RSSFeed {channel}
    }

    fn item_exists(&self, title: &str) -> bool {
        for item in self.channel.items() {
            if item.title.as_ref().unwrap() == title {
                return true;
            }
        }
        false
    }

    fn add_item(&mut self, title: &str, path: &str) {
        let url = BLOG_URL.to_owned() + path;
        let time = chrono::Utc::now();

        let mut item = Item::default();
        item.set_title(String::from(title));
        item.set_link(String::from(url));
        item.set_pub_date(String::from(time.to_rfc2822()));

        if self.item_exists(title) {
            self.remove_item(title);
        }

        let mut items = Vec::from(self.channel.items());
        items.push(item);
        self.channel.set_items(items);
    }

    fn remove_item(&mut self, title: &str) {
        let mut items: Vec<Item> = Vec::new();
        for item in self.channel.items() {
            if item.title.as_ref().unwrap() == title {
                continue;
            }
            items.push(item.clone());
        }

        self.channel.set_items(items);
    }

    fn save(&self) {
        std::fs::write(RSS_PATH, self.channel.to_string()).unwrap();
    }
}

#[derive(Serialize)]
struct Blog {
    #[serde(skip)]
    feed: RSSFeed,

    posts: HashMap<String, Post>,
}

impl Blog {
    fn new() -> Self {
        let mut blog = Blog {
            feed: RSSFeed::new(),
            posts: HashMap::new(),
        };

        blog.load_archive();
        blog
    }

    fn load_archive(&mut self) {
        let json_str = std::fs::read_to_string(POST_CACHE_PATH).unwrap();
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
        std::fs::write(POST_CACHE_PATH, json).unwrap();
    }

    fn remove_post(&mut self, post_title: &str) {
        self.feed.remove_item(post_title);
        self.posts.remove(post_title);
    }

    fn add_post(&mut self, path: &str) {
        let mut tera = Tera::default();
        tera.add_template_file(POST_TEMPLATE, Some("Post")).unwrap();

        let mut p = Post::init(path);
        let context = Context::from_serialize(&p).unwrap();

        let html = tera.render("Post", &context).unwrap();
        std::fs::write(HTML_DIRECTORY.to_owned() + &p.path, html).unwrap();

        self.feed.add_item(&p.title, &p.path);

        p.html = String::new();
        p.markdown = String::new();
        self.posts.insert(p.title.clone(), p);
    }

    fn build(&mut self) {
        let mut tera = Tera::default();
        tera.add_template_file(HOME_TEMPLATE, Some("Index")).unwrap();

        let mut context = Context::new();
        context.insert("posts", &self.posts);

        let html = tera.render("Index", &context).unwrap();
        std::fs::write(HTML_DIRECTORY.to_owned() + "index.html", html).unwrap();

        self.feed.save();
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

    check_paths();
    let mut blog = Blog::new();

    match args[1].as_str() {
        "publish" => blog.add_post(&args[2]),
        "remove"  => blog.remove_post(&args[2]),
        _ => {
            print_help();
            return;
        },
    };

    blog.build();
    blog.save_archive();
}
