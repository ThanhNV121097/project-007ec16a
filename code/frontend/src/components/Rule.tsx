/**
 * THE HAIRLINE: one pixel in --line across the full content width, and the
 * only thing on this site that separates one band from the next. No shaded
 * panels, no cards, no boxes — a rule, and then space.
 */
export default function Rule({ className = "" }: { className?: string }) {
  return <hr className={`h-px w-full border-0 bg-line ${className}`} />;
}
