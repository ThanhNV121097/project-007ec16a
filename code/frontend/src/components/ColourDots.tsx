import { type ReactNode } from "react";

/**
 * A product's colours as filled circles, the chosen one ringed. They are the
 * only colour on the page outside the photographs, which is why they are
 * small.
 *
 * The fill is the colour's own NAME handed to the browser — "navy", "olive",
 * "black" are all CSS colours — so a record names its colours in words an
 * owner would type and no hex is kept in the data. A name the browser does
 * not know draws the ring with no fill, and the word under the row still
 * says which is chosen.
 */
export default function ColourDots({ label, colours, value, onChange }: {
  label: ReactNode;
  colours: readonly string[];
  value: string;
  onChange: (colour: string) => void;
}) {
  if (colours.length === 0) return null;
  return (
    <fieldset data-block="variants" className="flex flex-col gap-2">
      <legend className="mb-2 text-xs uppercase tracking-wider text-ink-soft">
        {label}{value ? ` — ${value}` : ""}
      </legend>
      <div className="flex flex-wrap gap-2.5" role="radiogroup">
        {colours.map((c) => (
          <button
            key={c}
            type="button"
            role="radio"
            aria-checked={c === value}
            aria-label={c}
            onClick={() => onChange(c)}
            className={`size-5 rounded-full border border-line outline-offset-2 ${c === value ? "outline outline-1 outline-ink" : ""}`}
            style={{ backgroundColor: c.toLowerCase().replace(/\s+/g, "") }}
          />
        ))}
      </div>
    </fieldset>
  );
}
