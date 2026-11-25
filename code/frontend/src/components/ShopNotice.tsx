import { type ReactNode } from "react";

/**
 * The outline pill of the empty state's "Clear all" button. "Try again" wears
 * the very same classes, so the two read as one control wherever they appear.
 */
export const PILL =
  "inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink hover:bg-ink hover:text-ground";

/**
 * The block the shop's results area speaks in: the empty state, and the two
 * states the design never drew — the shoes on their way and the shoes that
 * could not be loaded. One band (`border-b border-line`), one top and bottom
 * spacing, everything left-aligned, so a visitor sees the same page whichever
 * of them it is.
 */
export default function ShopNotice({ className = "", children }: { className?: string; children: ReactNode }) {
  return <div className={`flex flex-col items-start gap-5 border-b border-line pb-16 pt-6 ${className}`}>{children}</div>;
}
