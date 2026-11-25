/**
 * Đôi's mark: two rings side by side, overlapping — a pair. Drawn in
 * currentColor so it follows the ink; public/favicon.svg is the same mark.
 * Deliberately nothing like three stripes or a trefoil: the shop sells
 * adidas and is not adidas.
 */
export default function LogoMark({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" className={className} aria-hidden="true" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="11" cy="16" r="8" />
      <circle cx="21" cy="16" r="8" />
    </svg>
  );
}
