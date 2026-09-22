---
title: "Notes on rotary position encoding"
date: 2026-09-21
---

All content from this article is taken from [(Su et al., 2021)](https://arxiv.org/pdf/2104.09864). I found the derivation of the paper's core idea a bit tricky, so I wanted to write down my understanding of its mathematics.

---

Since Transformers don't have an inherent notion of order, positional information needs to be encoded into the representations passed into the attention mechanism. Fixed sinuisodal encodings and learned encodings are two ways to do this, but what if we could construct a positional encoding that represents the absolute position of each representation, while also causing the dot product between two representations to depend on their relative positions in the sequence?

Consider two vectors, $\boldsymbol{z_1}$ and $\boldsymbol{z_2}$, represented in the complex plane. Their dot product is

$$
\boldsymbol{z}^T \boldsymbol{z_2} = r_1 r_2 (\cos \theta_1 \cos \theta_2 +\sin \theta_1 \sin \theta_2) = r_1 r_2 \cos(\theta_1 - \theta_2)
$$


If we encode a token's position by rotating its representation by an amount proportional to its position, the dot product between two position-encoded representations will depend on the difference between their positions.

Now let's consider how to rotate an representation, $\boldsymbol{x}$, by an amount proportional to its position, $m$, in the simplest case, where $\boldsymbol{x_m} \in \mathbb{R}^d$ and $d = 2$.

We know a 2D vector can be rotated by an angle $\phi$ using a rotation matrix.

$$
\boldsymbol{R} = \begin{bmatrix}
cos(\phi) & -sin(\phi) \\
sin(\phi) & cos(\phi)
\end{bmatrix}
$$

So $\boldsymbol{q} = \boldsymbol{R_q} \boldsymbol{W_q} \boldsymbol{x_m}$. The same concept applies to $\boldsymbol{k}$ as well.

Extending rotation to representations of an even dimension, we can divide the representation into $d / 2$ pairs. Each pair is treated as an independent 2D vector and is rotated independently. Each pair also uses different rotation frequencies, $\Theta = \\{ 10000 ^ {-2 i / d}, i \in \\{ 0, 1, 2, ... d / 2 - 1 \\} \\}$. So, a pair at $i$ in an representation at position $m$ is rotated by $m \Theta_i$ radians. A complete rotation can be written as a block-diagonal matrix, or as a sum of two Hadamard products for efficiency.

$$

\boldsymbol{R_q} \boldsymbol{q_m} =


\begin{bmatrix}
\text{cos}\,\Theta_0 m \\ \text{cos}\,\Theta_1 m \\ ... \\ \text{cos}\,\Theta_{d - 2} m \\ \text{cos}\,\Theta_{d - 1} m
\end{bmatrix}
\odot
\begin{bmatrix}
x_0 \\ x_1 \\ ... \\ x_{d - 2} \\ x_{d - 1}
\end{bmatrix}
+

\begin{bmatrix}
\text{sin}\,\Theta_0 m \\ \text{sin}\,\Theta_1 m \\ ... \\ \text{sin}\,\Theta_{d - 2} m \\ \text{sin}\,\Theta_{d - 1} m
\end{bmatrix}
\odot
\begin{bmatrix}
x_1 \\ -x_0 \\ ... \\ x_{d - 1} \\ -x_{d - 2}
\end{bmatrix}
$$

In summary, $ \boldsymbol{q^T k} = (\boldsymbol{R_q} \boldsymbol{W_q} \boldsymbol{x_m})^T (\boldsymbol{R_k} \boldsymbol{W_k} \boldsymbol{x_m}) $. Although queries and keys are rotated according to their absolute positions, their dot product contains a rotation determined by relative position $(n - m)$.

In practice, positional encodings are applied to several queries and keys at once, with rotation angles cached for efficiency. A naive implementation might look like the following.

```py
import torch

def rope_encoding(x):
  # B: batch size, N: number of embeddings, D: embedding dimension
  B, N, D = x.shape

  theta = 10000 ** ((-2 * torch.arange(D / 2)) / D)
  theta = torch.repeat_interleave(theta, repeats=2)
  angles = torch.arange(N)[:, None] * theta

  # These can be cached
  cos_angles, sin_angles = torch.cos(angles), torch.sin(angles)
  signs = ((-1) ** torch.arange(1, D + 1)).expand(N, -1)
  alt_idx = torch.arange(D + 1)[torch.arange(D) ^ 1]

  return x * cos_angles + x[:, :, alt_idx] * signs * sin_angles
```

This construction offers a few more interesting properties that makes it better than other positional encoding schemes:

- Rotations preserve the rotation of a vector. They also preserve the angle between two vectors when the same rotation is applied to both, so positional information is introduced by changing the direction of the representation rather than its magnitude.

- The positional encoding is periodic. However, this isn't a problem in practice since multiple frequencies are used simultaneously. The lower indexed pairs rotate more quickly, while higher indexed pairs rotate more slowly, providing positional information at multiple scales and making each position encoding unique across massive context lengths.

- As the relative distance between two tokens increaes, the rotations associated with the different dimensions becomes increasingly misaligned. When their contributions are combined, the encoded position information in the attention scores tend to weaken with distance, adding a locality bias.