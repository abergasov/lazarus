<script lang="ts">
  let { points, width = 80, height = 24, color = 'var(--blue, #007AFF)' }: {
    points: number[];
    width?: number;
    height?: number;
    color?: string;
  } = $props();

  const path = $derived.by(() => {
    if (!points || points.length < 2) return '';
    const min = Math.min(...points);
    const max = Math.max(...points);
    const range = max - min || 1;
    const step = width / (points.length - 1);
    return points.map((v, i) => {
      const x = i * step;
      const y = height - ((v - min) / range) * (height - 4) - 2;
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
  });
</script>

<svg {width} {height} viewBox="0 0 {width} {height}" fill="none">
  <path d={path} stroke={color} stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
</svg>
