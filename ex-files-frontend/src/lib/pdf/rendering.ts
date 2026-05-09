export type RenderedCluster<TItem extends RenderedItem> = {
    id: string;
    x: number;
    y: number;
    items: TItem[];
};

export type RenderedItem = {
    id: string;
    x: number;
    y: number;
}

export function deriveClusters<TItem extends RenderedItem>(items: TItem[], threshold: number): RenderedCluster<TItem>[] {
    const out: RenderedCluster<TItem>[] = [];
    for (const c of items) {
        const hit = out.find((cl) => {
            const dx = cl.x - c.x;
            const dy = cl.y - c.y;
            return dx * dx + dy * dy <= threshold * threshold;
        });

        if (hit) {
            hit.items.push(c);
            const n = hit.items.length;
            hit.x = hit.items.reduce((s, k) => s + k.x, 0) / n;
            hit.y = hit.items.reduce((s, k) => s + k.y, 0) / n;
        } else {
            out.push({ id: c.id, x: c.x, y: c.y, items: [c] });
        }
    }
    return out;
}
