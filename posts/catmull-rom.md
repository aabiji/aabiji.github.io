# Catmull-Rom splines

Suppose we want to draw a curved line that passes through a set
of xy coordinates. We can use a Catmull-Rom spline to solve the problem,

But first of all, what's a spline? A spline is a mathematical tool for
creating smooth, continuous curves through or near a set of points,
often by stitching together cubic Bézier curves. A Bézier curve is
a parametric, piecewise polynomial curve defined by control points
that influence its shape. The curve is computed by interpolating
the control points points using a parameter t∈[0,1],
generating the curve as a weighted combination of the control points.
In a regular spline, control points are manually defined and not
part of the resulting curve, but in a Catmull-Rom spline, the
curve passes directly through the supplied points, automatically
determining its shape. My explanation is terrible, so here's a [better
one](https://www.youtube.com/watch?v=jvPPXbo87ds).

Catmull-Rom splines are defined by 4 control points; p₀, p₁, p₂, p₃,
t∈[0,1], which represents the distance between the start and end
of the curve, and α∈[0,1], which influences how the distances between
the control points affect the resulting curve. [TODO: include diagram and wikipedia diagram]
The curve is generated between p₁, p₂ (p₀ and p₃ will not be part of the curve).
So when t = 0, the interpolated point is p₁ and when t = 1, the interpolated point
is p₂. 
