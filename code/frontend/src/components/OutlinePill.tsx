import { type ReactNode } from "react";
import { Link } from "react-router-dom";
import Icon from "./Icon";

/**
 * THE ONE CALL TO ACTION: an outlined pill, small letterspaced capitals, an
 * arrow. Never filled — nothing on this site is filled with colour — and the
 * same on every page, so a visitor learns once what can be pressed.
 */
export default function OutlinePill({ to, children, testid, className = "" }: {
  to: string;
  children: ReactNode;
  testid?: string;
  className?: string;
}) {
  return (
    <Link
      to={to}
      data-testid={testid}
      className={`inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink transition-colors hover:bg-ink hover:text-ground ${className}`}
    >
      {children}
      <Icon name="arrow" size={14} />
    </Link>
  );
}
