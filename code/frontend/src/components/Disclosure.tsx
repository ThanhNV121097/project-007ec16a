import { type ReactNode } from "react";

/**
 * A DISCLOSURE ROW: a label at the left, a + at the right, a hairline under
 * it — and what a buyer asks next behind it. Three of them stacked carry
 * everything a record's page has to say without the page becoming a column
 * of paragraphs.
 *
 * A row with nothing in it is not drawn: a + that opens onto nothing is a
 * broken control.
 */
export default function Disclosure({ rows, className = "" }: {
  rows: readonly { label: ReactNode; content: ReactNode }[];
  className?: string;
}) {
  const shown = rows.filter((r) => r.content !== null && r.content !== undefined && r.content !== false && r.content !== "");
  if (shown.length === 0) return null;
  return (
    <div data-block="details" className={`border-t border-line ${className}`}>
      {shown.map((r, i) => (
        <details key={i} className="group border-b border-line">
          <summary className="flex cursor-pointer list-none items-center justify-between py-3.5 text-sm text-ink [&::-webkit-details-marker]:hidden">
            {r.label}
            <span aria-hidden="true" className="text-lg leading-none text-ink-soft transition-transform group-open:rotate-45">+</span>
          </summary>
          <div className="pb-5 text-sm text-ink-soft">{r.content}</div>
        </details>
      ))}
    </div>
  );
}
