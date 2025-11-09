# Simplifying polylines to exactly *n* points
[Abigail Adegbiji](https://aabiji.github.io/) • September 28, 2025

Plotting big datasets often results in unreadable graphs full of noise.
What's needed is a simplfied version that preserves the overall shape by
reducing the number of points to something manageable.

One popular algorithm for this is the
[Ramer-Douglas-Peuker algorithm](https://en.wikipedia.org/wiki/Ramer%E2%80%93Douglas%E2%80%93Peucker_algorithm), But it doesn't
let you control the exact number of output points, instead it removes
points based on a distance threshold (ε). To reach
a specific number of points, the algorithm has to be ran multiple
times, (doing binary search over ε), which is inefficient.

A better approach is the [Visvalingam-Whyatt algorithm](https://en.wikipedia.org/wiki/Visvalingam%E2%80%93Whyatt_algorithm).
It ranks points by how little they contribute to the shape of the line, then removes the least important ones first. Specifically, it calculates the area of the triangle formed by each point and its two neighbors, removes the point with the smallest triangle, and repeats until only n points remain. The naive version runs in O(n²), but with clever use of a min heap to track triangle areas efficiently, the runtime drops to O(n log n).

First we need to implement the min heap. It'll store triangle areas, allowing constant time access to the smallest area and logarithmic time updates.
```ts
export class MinHeap {
  values: any[];
  smallerThan: (a: any, b: any) => boolean;

  constructor(cmp: (a: any, b: any) => boolean) {
    this.values = [];
    this.smallerThan = cmp;
  }

  empty(): boolean {
    return this.values.length === 0;
  }

  private swap(a: number, b: number) {
    const temp = this.values[a];
    this.values[a] = this.values[b];
    this.values[b] = temp;
  }

  insert(value: any) {
    this.values.push(value);
    let index = this.values.length - 1;

    // moving up the tree, swap nodes that violate the min heap property
    while (index > 0) {
      const parent = Math.floor((index - 1) / 2);
      if (this.smallerThan(this.values[parent], this.values[index])) break;
      this.swap(index, parent);
      index = parent;
    }
  }

  pop(): any {
    if (this.values.length === 0) return null;

    const removed = this.values[0];
    const last = this.values.pop();
    if (this.values.length > 0 && last !== undefined) {
      this.values[0] = last;

      // heapify down
      let index = 0;
      while (true) {
        const left = 2 * index + 1;
        const right = 2 * index + 2;
        let smallest = index;

        if (left < this.values.length &&
          this.smallerThan(this.values[left], this.values[smallest]))
          smallest = left;

        if (right < this.values.length &&
          this.smallerThan(this.values[right], this.values[smallest]))
          smallest = right;

        if (smallest === index) break;

        this.swap(index, smallest);
        index = smallest;
      }
    }

    return removed;
  }
}
```

Now we can implement the algorithm.
We'll first need a function to compute the area of a triangle given three points:

```ts
type Vec2 = { x: number, y: number };
const getArea = (a: Vec2, b: Vec2, c: Vec2): number =>
  0.5 * Math.abs(
    a.x * b.y + b.x * c.y + c.x * a.y -
    a.x * c.y - b.x * a.y - c.x * b.y
  );
```

Next, define the data structure that will be stored in the heap:

```ts
type HeapValue = {
  area: number,
  indexA: number,
  indexB: number,
  indexC: number,
};

const cmpHeapValues = (a: HeapValue, b: HeapValue): boolean => a.area < b.area;
```

A helper function is needed to find the nearest non-deleted point in a given direction:

```ts
const nearestPoint = (data: (Vec2 | null)[], index: number, direction: number): number => {
  while (data[index] === null) index += direction;
  return index;
};
```

Now for the core of the algorithm:

```ts
function visvalingamWhyattAlgorithm(data: Vec2[], targetLength: number): Vec2[] {
  const heap = new MinHeap(cmpHeapValues);

  let arr = [...data] as (Vec2 | null)[];
  let removedCount = 0;

  // populate the heap with initial triangle areas
  for (let i = 1; i < arr.length - 1; i++) {
    const area = getArea(arr[i - 1]!, arr[i]!, arr[i + 1]!);
    heap.insert({ area, indexA: i - 1, indexB: i, indexC: i + 1 });
  }

  while (!heap.empty() && removedCount < data.length - targetLength) {
    let value = heap.pop() as HeapValue;

    // skip values where any point has already been removed
    while (!arr[value.indexA] || !arr[value.indexB] || !arr[value.indexC]) {
      if (heap.empty()) break;
      value = heap.pop();
    }

    // memove the middle point
    arr[value.indexB] = null;
    removedCount++;

    // recalculate area for the new triangle to the left
    const prev = nearestPoint(arr, value.indexA - 1, -1);
    if (prev >= 0) {
      const [a, b, c] = [prev, value.indexA, value.indexC];
      const newArea = getArea(toVec2(arr[a]!), toVec2(arr[b]!), toVec2(arr[c]!));
      heap.insert({ area: newArea, indexA: a, indexB: b, indexC: c });
    }

    // Recalculate area for the new triangle to the right
    const next = nearestPoint(arr, value.indexC + 1, 1);
    if (next < arr.length) {
      const [a, b, c] = [value.indexA, value.indexC, next];
      const newArea = getArea(toVec2(arr[a]!), toVec2(arr[b]!), toVec2(arr[c]!));
      heap.insert({ area: newArea, indexA: a, indexB: b, indexC: c });
    }
  }

  // remove deleted points
  return arr.filter(p => p !== null) as Vec2[];
}
```

Now we can efficiently reduce the dataset by removing points that lend the least perceptible change.
