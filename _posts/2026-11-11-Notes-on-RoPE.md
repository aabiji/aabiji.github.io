---
title: "Notes on RoPE"
date: 2026-11-11
---

Transformers don't have an inherent notion of order, so positional information needs to be encoded somewhere. Fixed sinusoidal encodings and learned encodings do this by adding a position vector to the input embeddings [(Vaswani et al., 2017)](https://arxiv.org/pdf/1706.03762). Rotary Position Embedding (RoPE) [(Su et al., 2021)](https://arxiv.org/pdf/2104.09864) instead encodes position directly into the attention score, so that the score between a query at position $m$ and a key at position $n$ depends on their contents and on their offset $n - m$. The following derivation is based off of Su et al.

The idea starts with a fact about 2D vectors. Take two vectors, $v_1$ and $v_2$ with lengths $r_1, r_2$ and angles $\theta_1, \theta_2$. Their dot product will be

$$
\begin{aligned}
& \langle r_1 \cos \theta_1 , r_1 \sin \theta_1  \rangle \cdot \langle r_2 \cos \theta_2 , r_2 \sin \theta_2 \rangle \\
&= (r_1 \cos \theta_1) (r_2 \cos \theta_2) + (r_1 \sin \theta_1) (r_2 \sin \theta_2) \\
&= r_1 r_2 (\cos \theta_1 \cos \theta_2 + \sin \theta_1 \sin \theta_2) \\
&= r_1 r_2 \cos(\theta_1 - \theta_2)
\end{aligned}
$$

which depends on the difference of their angles. So the similarity between two vectors rotated by an amount proportional to their positions will depend on the difference of positions.

This idea can also be expressed using matrices. A rotation by angle $\phi$ can be expressed as the matrix

$$ \boldsymbol{R} = \begin{bmatrix} \cos(\phi) & -\sin(\phi) \\ \sin(\phi) & \cos(\phi) \end{bmatrix} $$

and it has two needed properties. Angles add, $R(a)R(b) = R(a + b)$, and the transpose is the inverse, $R(a)^T = R(-a)$. Together these give

$$R(a)^TR(b) = R(b - a).$$

We can apply this concept to attention. $q_m = W_q  x_m$ and $k_n = W_k  x_n$ are the usual projections of the tokens at positions $m$ and $n$. For a single 2D pair and some constant $\theta$, RoPE rotates each vector by an angle proportional to its position,

$$
\tilde q_m = R(\theta m) \, q_m \qquad \tilde k_n = R(\theta n) \, k_n
$$

so the attention score becomes

$$
\begin{aligned}
&\tilde q_m^T \tilde k_n \\
&= q_m^T \, R(\theta m)^T \, R(\theta n) k_n \\
&= q_m^T R(\theta[n - m]) k_n
\end{aligned}
$$

Note that the two rotations have to be different for this to work, otherwise $R^T R = I$ and position info would disappear entirely.

To handle a vector of a higher even dimension $d$, split it into $d / 2$ pairs and rotate each independently, giving pair $i$ the frequency

$$ \Theta_i = 10000 ^ {-2 i / d}, \qquad i = 0, 1, \dots d / 2 - 1 $$

similar to what's used in sinusoidal encodings. A vector at position $m$ is rotated by a block-diagonal matrix $R_m$ whose blocks are $R(\Theta_i m)$. Transposes and products of block-diagonal matrices act block by block, so $R_m^T R_n = R_{n - m}$ still holds and the score becomes

$$ \tilde q_m^T \tilde k_n = q_m^T R_{n-m}\, k_n = \sum_{i=0}^{d/2-1} q_{m,i}^T\, R(\Theta_i [n-m]) \, k_{n,i} $$

where $q_{m, i}$ and $k_{n, i}$ are the $i$-th pair. Each pair contributes a term oscillating in $n - m$ at its own frequency. Lower pair indices rotate quickly and resolve small offsets while higher pair indices rotate more slowly and resolve large offsets. A single pair is periodic, but combining many frequencies keeps offsets distinct enough to be distinguishable over long ranges.

In practice one would never create the full matrix because applying $R_m$ to a vector results in two element-wise products.

$$
R_m q_m =

\begin{bmatrix}
\text{cos}(\Theta_0 m) \\ \text{cos}(\Theta_0 m) \\ \vdots \\ \text{cos}(\Theta_{d/2 - 1} m) \\ \text{cos}(\Theta_{d/2 - 1} m)
\end{bmatrix}
\odot
\begin{bmatrix}
q_0 \\ q_1 \\ \vdots \\ q_{d - 2} \\ q_{d - 1}
\end{bmatrix}
+

\begin{bmatrix}
\text{sin}(\Theta_0 m) \\ \text{sin}(\Theta_0 m) \\ \vdots \\ \text{sin}(\Theta_{d/2 - 1} m) \\ \text{sin}(\Theta_{d/2 - 1} m)
\end{bmatrix}
\odot
\begin{bmatrix}
-q_1 \\ q_0 \\ \vdots \\ -q_{d - 1} \\ q_{d - 2}
\end{bmatrix}
$$

RoPE is applied to the projected queries and keys once per attention layer, not to the input embeddings or the values. Rotation angles only depend on position so they can be cached. A naive implementation might look like the following.

```py
import torch

def rope(x):
  _, N, D = x.shape # (Batch Size, Embeddings, Embedding Dimension)

  theta = 10000 ** ((-2 * torch.arange(D // 2)) / D)
  angles = torch.arange(N)[:, None] * theta
  cos = angles.cos().repeat_interleave(2, dim=-1)
  sin = angles.sin().repeat_interleave(2, dim=-1)

  x_rotated_pairs = torch.stack((-x[..., 1::2], x[..., 0::2]), dim=-1).flatten(-2)
  return x * cos + x_rotated_pairs * sin
```

I think RoPE has some interesting implications. Since it's applied inside every attention layer, positional information is refreshed at every layer, instead of being applied once to input embeddings and needing to be strong enough to survive through the residual stream. It's possible that the model also doesn't need to spend its capacity learning relative offsets from absolute positions. Lastly, RoPE can be evaluated at any position, but model performance degrades when evaluated on sequences past the training length because the model sees angles it never trained on. Methods like YaRN [(Peng et al., 2023)](https://arxiv.org/pdf/2309.00071) adjust frequencies so that longer sequences fit in the trained range, and then a little fine-tuning recovers quality.