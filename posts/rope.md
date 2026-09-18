# Rotary Positional Encoding For Dummies
[Abigail Adegbiji](https://aabiji.github.io/) • September 17, 2026

* High level premise and intuition

* Proof (explanation -> reorder steps to seem more intuitive, maybe shouldn't rely on having observations):

- f turns embeddings into queries or keys by projecting and adding positional info based off of position
- g computes attention scores and adds positional info based off of relative displacement between queries/keys at different positions

- Now let's examine the example of 2d vectors since it's the simplest and easiest to visualize. There are 3 main ways to represent a 2d value:
  - As a polar coordinate
  - As a complex number in rectangular form (show the trig equivalent)
  - As a complex number in polar form

- Notice an interesting property: multiplying two complex numbers in polar form gives us a way to compute an inner product and encode relative positional info (show where cos(theta_\1 - theta_\2) is coming from.

- Now, consider the case where m = n. There would be no displacement so the output of g should have no positional info encoded.

- Notice an interesting property: When m = 0, f should encode no positional info on the input vector, so all properties of that vector should depend solely on the vector itself. Remember that n could be any integer between [0, L] where L is the sequence length, so what holds at n = 0 should hold everywhere else. Therefore, properties of the queries/keys should only depend on themselves at all positions. <show that the radial and angular only depend on vector contents>

- So, we'll need some way to add positional info back in, let's call that function phi(m)


