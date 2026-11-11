---
title: "Notes on rotary position encoding"
date: 2026-09-21
---

Since Transformers don't have an inherent notion of order, positional information needs to be encoded somwhere. Fixed sinuisodal encodings and learned encodings do this by adding a position vector to the input embeddings [(Vaswani et al., 2017)](https://arxiv.org/pdf/1706.03762). Rotary Position Embedding (RoPE) [(Su et al., 2021)](https://arxiv.org/pdf/2104.09864) instead encodes position directly into the attention score itself, so that the score between a query at position $m$ and a key at position $n$ depends on their contents and on their offset $n - m$. The intuition and derivation below follows Su et al.

---

Take two 2D vectors, $v_1$ and $v_2$ with lengths $r_1, r_2$ and angles $\theta_1, \theta_2$ .The angle for their dot product will be $\theta_1 - \theta_2$.

$$
\begin{aligned}
& \langle r_1 \cos \theta_1 , r_1 \sin \theta_1  \rangle \cdot \langle r_2 \cos \theta_2 , r_2 \sin \theta_2 \rangle \\
&= (r_1 \cos \theta_1) (r_2 \cos \theta_2) + (r_1 \sin \theta_1) (r_2 \sin \theta_2) \\
&= r_1 r_2 (\cos \theta_1 \cos \theta_2 + \sin \theta_1 \sin \theta_2) \\
&= r_1 r_2 \cos(\theta_1 - \theta_2)
\end{aligned}
$$

So the dot product depends on the difference of the angles. If we rotate each vector by an amount proportional to its position, that difference will contain the difference of positions.

This same concept applies to rotation matrices. A 2D rotation by $\theta_0$ is the matrix

$$ \boldsymbol{R} = \begin{bmatrix} cos(\phi) & -sin(\phi) \\ sin(\phi) & cos(\phi) \end{bmatrix} $$

with the important properties. Angles add, $R(a)R(b) = R(a + b)$, and the transpose of a rotation is a rotation by its inverse, $R(a)^T = R(-a)$, $R(a)^TR(b) = R(b - a)$.

Now consider attention. If we rotate queries and keys by their positions in the token sequence, we can encode the relative distance between positions in the attention scores.

$$
q = R_m W_q \, x_m \\
k = R_n W_k \, x_n \\
q^T k = (R(\theta m)\,x_m)^T (R(\theta n)\,x_n) \\
= x_m^T \, R(\theta m) \, R(\theta n) \, x_n \\
= x_m^T R_{n - m} x_n \\
$$

To rotate vectors of dimension $d$ in practice, they are split into $d / 2$ pairs and rotated independelty. Each pair uses different rotation frequencies, $\Theta = \\{ 10000 ^ {-2 i / d}, i \in \\{ 0, 1, 2, ... d / 2 - 1 \\} \\}$, similar to what was used in [(Vaswani et al., 2017)](https://arxiv.org/pdf/1706.03762). So, a pair at $i$ in the representation at position $m$ is rotated by $m \Theta_i$ radians. A complete rotation can be written as a block-diagonal matrix, or as a sum of two element-wise vector multiplications for efficiency.

$$

\boldsymbol{R_m} \boldsymbol{q_m} =

\begin{bmatrix}
\text{cos}(\Theta_0 m) \\ \text{cos}(\Theta_0 m) \\ ... \\ \text{cos}(\Theta_{d - 1} m) \\ \text{cos}(\Theta_{d - 1} m)
\end{bmatrix}
\odot
\begin{bmatrix}
q_0 \\ q_1 \\ ... \\ q_{d - 2} \\ q_{d - 1}
\end{bmatrix}
+

\begin{bmatrix}
\text{sin}(\Theta_0 m) \\ \text{sin}(\Theta_0 m) \\ ... \\ \text{sin}(\Theta_{d - 1} m) \\ \text{sin}(\Theta_{d - 1} m)
\end{bmatrix}
\odot
\begin{bmatrix}
-q_1 \\ q_0 \\ ... \\ -q_{d - 1} \\ q_{d - 2}
\end{bmatrix}
$$

Positional encodings are applied to several queries and keys at once, with rotation angles cached for efficiency. A naive implementation might look like the following.

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

ROPE has some interesting implications. Since position info is included in the attention function, positionality is included at every layer. On the other hand, positional info in absolute and relative position encodings need to be strong enough to not get diluted with depth. Second, this means that the model doesn't need to spend part of its learning capacity on extracting relative relationships between tokens. Second, because ROPE is a deterministic function, it has no hard length cap. So with appropriate fine tuning, the model can process sequences of varying lengths. Lastly, with the standard frequency schedule, the contribution between distant tokens tends to decay with distance, which adds a reasonable inductive bias for language modelling.

