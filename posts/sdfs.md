# Notes on signed distance functions
[Abigail Adegbiji](https://aabiji.github.io/) • February 22, 2026

Signed Distance Functions are really cool. They're functions that take
a point in space and return the shortest signed distance to the nearest surface.
Positive if outside the shape, negative if inside, and zero if exactly on the surface.

Using the classic sphere example:

$f(x, y, z) = \sqrt{x^2 + y^2 + z^2} - r$

Where $(x, y, z)$ is the point being tested, and $r$ is the sphere's radius,
centered at (0, 0, 0) for simplicity.

Combined with ray marching, SDFs are very powerful rendering tools. Instead of
solving ray-shape intersections analytically, we march along the ray, using the
SDF to tell us how far we can safely jump without missing anything. In the best case,
a ray heading straight at an isolated object converges in very few, very large steps.

```c
vec3 raymarch(vec3 camera_position, vec3 ray_direction) {
    // These constants are arbitrary
    const float max_steps = 64;
    const float max_distance = 200;
    const float epsilon = 0.001;

    float total_distance = 0;
    vec3 ray_origin = camera_position;

    for (int i = 0; i < max_steps; i++) {
        float distance = SDF(ray_origin);
        if (distance < epsilon) {
            return ray_origin; // Hit a surface
        }
        ray_origin += ray_direction * distance;

        total_distance += distance;
        if (total_distance > max_distance) break;
    }
    return vec3(0.0); // Didn't hit anything
}
```

Each fragment gets its own ray. The ray's direction is just:
```c
// Convert the on screen coordinate to UV coordinates, in the range of -1 to 1
// Dividing by resolution.y preserves the aspect ratio.
vec2 uv = (gl_FragCoord - 0.5 * resolution) / resolution.y;

// Map a ray pointing into the scene to camera space.
vec3 ray_direction = normalize(mat3(view_matrix) * vec3(uv, -1.0));
```

Now for the actual SDF itself. We can store all the object data, like position and size
in a uniform buffer that we can access in the fragment shader. Then in the
SDF function, we just find the surface that's closest to the ray's point. This way,
we can render multiple objects without needing instancing.

```c
// `SphereData`, `num_spheres` and `uniform_spheres_data` are defined up here...

// Example that draws spheres:
float scene_SDF(vec3 point) {
    float min_dist = 999.9f;
    for (int i = 0; i < num_spheres; i++) {
        SphereData sphere = uniform_spheres_data[i];
        float distance = length(point - sphere.position) - sphere.radius;
        min_dist = min(distance, min_dist);
    }
    return min_dist;
}
```

Once we hit a surface, we need a normal for lighting. The SDF's gradient gives us exactly that.
In general, the gradient of a function $f$ at a point $(x, y, z)$ tells you what direction to move
in from the point to most quickly increase the value of $f$. Near the surface of an SDF, the fastest
way to increase the value (from negative to positive or zero to more positive), is to move straight
outwards, perpendicular to the surface.

We can approximate the gradient using tiny offsets:
```c
vec3 gradient(vec3 p) {
    float epsilon = 0.001;
    return normalize(vec3(
        scene_SDF(vec3(p.x + epsilon, p.y, p.z)) - scene_SDF(vec3(p.x - epsilon, p.y, p.z)),
        scene_SDF(vec3(p.x, p.y + epsilon, p.z)) - scene_SDF(vec3(p.x, p.y - epsilon, p.z)),
        scene_SDF(vec3(p.x, p.y, p.z + epsilon)) - scene_SDF(vec3(p.x, p.y, p.z - epsilon)),
    ));
}
```

From here we can proceed to do the rest of our lighting calculations, perhaps some basic Phong shading:
```c
vec3 ambient = ambient_color * light_color;

vec3 light_dir = normalize(light_pos - world_pos);
vec3 diff = max(dot(normal, light_dir), 0.0);
vec3 diffuse = diffuse_color * diff * light_color;

vec3 view_dir = normalize(view_pos - world_pos);
vec3 halfway = normalize(light_dir + view_dir);
float spec = pow(max(dot(normal, halfway), 0.0), shininess);
vec3 specular = specular_color * spec * light_color;

fragColor = vec4(ambient + diffuse + specular, 1.0);
```
