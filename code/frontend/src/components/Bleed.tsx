import type { ReactNode } from "react";

/**
 * WHAT RUNS OFF THE EDGE OF THE SCREEN. Words keep the page's 1200px
 * measure; the photographs and the panel where a visitor acts — the bag's
 * total, the chart, the form — do not. <Bleed> takes them from the column
 * they sit in out to the edge:
 *
 *   side="right"  to the right edge from 1024px, to both edges below it
 *   side="left"   to the left edge from 1024px, to both edges below it
 *   side="both"   to both edges at every size
 *
 * `ground` paints the photographs' own pale grey (--surface) behind it and
 * pads the inside by as much as it reached, so the words in it stay on the
 * page's grid while the grey runs out. Without it the child itself — a
 * photograph — meets the edge.
 *
 * It widens by negative margins (--bleed in theme.css), which a block with
 * no width of its own, or a stretched item of a CSS grid, turns into width:
 * so it can BE a grid's column (`className="lg:col-span-5"`), or sit inside
 * a flex column that has a width of its own. Vertical padding is the
 * caller's, through className.
 */
const REACH = {
  right: "-ml-[var(--bleed)] -mr-[var(--bleed)] lg:ml-0",
  left: "-ml-[var(--bleed)] -mr-[var(--bleed)] lg:mr-0",
  both: "-ml-[var(--bleed)] -mr-[var(--bleed)]",
} as const;

const PAD = {
  right: "bg-surface pl-[var(--bleed)] pr-[var(--bleed)] lg:pl-8",
  left: "bg-surface pl-[var(--bleed)] pr-[var(--bleed)] lg:pr-8",
  both: "bg-surface pl-[var(--bleed)] pr-[var(--bleed)]",
} as const;

export default function Bleed({ side = "right", ground = false, as: Tag = "div", className = "", children }: {
  side?: keyof typeof REACH;
  ground?: boolean;
  as?: "div" | "aside" | "section";
  className?: string;
  children: ReactNode;
}) {
  return (
    <Tag data-bleed={side} className={`${REACH[side]} ${ground ? PAD[side] : ""} ${className}`}>
      {children}
    </Tag>
  );
}
