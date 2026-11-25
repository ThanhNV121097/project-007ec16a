import { type ReactNode } from "react";
import { Link } from "react-router-dom";

export interface ColourChoice {
  id: string;
  name: string;
  /** A CSS colour name kept on the record ("black", "pink"), never a hex. */
  swatch?: string;
}

/**
 * COLOUR DOTS: the colourways of one model as filled circles about 14px
 * across, the one being looked at carrying a thin ring. Each dot is a link
 * to that colourway's own page — the choice is real, not decoration. A model
 * with one colourway shows its one dot, ringed.
 */
export default function ColourDots({ label, items, current }: {
  label: ReactNode;
  items: readonly ColourChoice[];
  current: string;
}) {
  if (items.length === 0) return null;
  return (
    <div data-block="variants" className="flex flex-col gap-1.5">
      <p className="text-xs uppercase tracking-wider text-ink-soft">{label}</p>
      <ul className="-ml-2 flex flex-wrap">
        {items.map((c) => {
          const on = c.id === current;
          return (
            <li key={c.id}>
              <Link to={`/shop/${c.id}`} aria-label={c.name} aria-current={on ? "page" : undefined} title={c.name}
                className="grid size-9 place-items-center">
                <span
                  className={`block size-3.5 rounded-full border border-line outline-offset-2 transition-transform hover:scale-110 ${on ? "outline outline-1 outline-ink" : ""}`}
                  style={{ backgroundColor: c.swatch }}
                />
                <span className="sr-only">{c.name}</span>
              </Link>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
