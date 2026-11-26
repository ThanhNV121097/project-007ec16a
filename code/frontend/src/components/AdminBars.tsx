/**
 * A small bar chart for the overview, in the management area's own blue:
 * columns for figures over time (sales a day), rows for figures by kind
 * (pairs sold by line). Every bar carries its figure as text, so the chart
 * is read, not guessed at.
 */
export interface Bar {
  label: string;
  value: number;
  display: string;
}

export default function AdminBars({ rows, kind = "columns", caption }: {
  rows: readonly Bar[];
  kind?: "columns" | "rows";
  caption: string;
}) {
  const max = Math.max(1, ...rows.map((r) => r.value));
  if (kind === "rows") {
    return (
      <figure>
        <figcaption className="sr-only">{caption}</figcaption>
        <ul className="flex flex-col gap-3">
          {rows.map((r) => (
            <li key={r.label} className="grid grid-cols-3 items-center gap-3 text-sm sm:grid-cols-5">
              <span className="min-w-0 break-words text-ink-soft">{r.label}</span>
              <span className="col-span-1 h-2 rounded-sm bg-wash sm:col-span-3">
                <span className="block h-full rounded-sm bg-accent" style={{ width: `${(r.value / max) * 100}%` }} />
              </span>
              <span className="text-right tabular-nums text-ink">{r.display}</span>
            </li>
          ))}
        </ul>
      </figure>
    );
  }
  return (
    <figure>
      <figcaption className="sr-only">{caption}</figcaption>
      <div className="flex h-48 items-end gap-1.5">
        {rows.map((r) => (
          <div key={r.label} className="flex h-full flex-1 flex-col justify-end" title={`${r.label}: ${r.display}`}>
            <span className={`block w-full rounded-sm ${r.value > 0 ? "bg-accent" : "bg-wash"}`} style={{ height: `${Math.max(2, (r.value / max) * 100)}%` }} />
          </div>
        ))}
      </div>
      <div className="mt-2 flex gap-1.5 border-t border-line pt-2 text-[0.625rem] text-ink-soft">
        {rows.map((r) => <span key={r.label} className="flex-1 text-center tabular-nums">{r.label}</span>)}
      </div>
    </figure>
  );
}
