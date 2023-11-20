# Introducing ```focus```

Even though I love what I work on, I procrastinate a lot -- too much.
So naturally, in a bid to solve my problem, I decided to write a program that
helps me be more productive. Not only would it make me productive, but it would
make me more familiar with day-to-day Rust.

Focus is 3 things integrated into 1 app:
- It's a todo list
  The key difference between my list and other todo lists is the fact that the
  tasks are hierecal. You can define sub-tasks of sub tasks of tasks. I prefer
  this because it allows me to break down a "huge" task into a series of
  actionable sub-tasks.
- It's a music player
  I like listening to music (without lyrics) while I work. Having music in the
  background sets a nice ambience and helps me focus.
- It's a website blocker
  This is probably the most important feature. From the moment I hit the play
  button, it blocks all the sites I've defined in the json config. When I try to 
  go to any of the sites, it redirects me to a page on localhost saying "Wanna
  watch grass grow?" This reminds me to get back on track and is lighthearted
  enough that I don't get bummed out about it.

I wrote focus in Rust using the egui and rodio crates.
I chose egui because it's an immediate mode ui. So, even though layouts might be
a pain, the code would be a lot more simple.
Instead of using rodio, I originally planned on using symphonia and cpal
together, but then I saw that rodio made my life so much easier.
Some thoughts on the project:
- As a Rust noob, writing a tree data structure was non trivial
  At first I tried a regular tree with pointers, but that was wrought with
  borrowing errors. Throwing a Rc<RefCell<T>> didn't solve the problem.
  Eventually, I pivoted away from pointers and instead had a vector of nodes
  with nodes referencing child and parent nodes through indices.
  At first I was salty that I had to resort to that, then I realized that Rust
  forced me to rethink my solution in a way that way inherently memory safe
