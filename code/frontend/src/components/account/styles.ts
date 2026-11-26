/**
 * The classes the customer-account screens share, so a field, a pill and a
 * link are the same on all three pages. They are the design's own parts
 * (design-system.md 2.1.5, 2.1.6 and 2.1.16) written with its tokens, plus
 * the two things those parts leave out and a keyboard needs: a visible focus
 * outline, and a 44px hit area (min-h-11).
 */
export const FOCUS = "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink";

/** The text field of the checkout: a field-line edge on the ground, ink on focus. */
export const FIELD = `w-full appearance-none rounded-sm border border-field bg-ground px-3 py-3 text-base text-ink placeholder:text-ink-soft focus:border-ink ${FOCUS} sm:text-sm`;

export const LABEL = "mb-1.5 block text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft";

/** The filled pill: the one commit action of a view. */
export const FILLED = `inline-flex min-h-11 items-center justify-center gap-2 self-start rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85 disabled:opacity-60 ${FOCUS}`;

/** The outline pill, as a button: the page's other actions. */
export const OUTLINE = `inline-flex min-h-11 items-center justify-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink transition-colors hover:bg-ink hover:text-ground disabled:opacity-60 disabled:hover:bg-transparent disabled:hover:text-ink ${FOCUS}`;

/** A link inside a sentence. */
export const TEXT_LINK = `text-ink underline underline-offset-4 ${FOCUS}`;
