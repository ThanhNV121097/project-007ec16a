import { useMemo } from "react";
import { qrModules } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";

/** The light margin a scanner needs round the code, in modules. */
const QUIET = 4;

/** One path of unit squares for every dark module, shifted in by the margin. */
function pathOf(rows: boolean[][]): string {
  const parts: string[] = [];
  rows.forEach((row, r) => {
    row.forEach((dark, c) => {
      if (dark) parts.push(`M${c + QUIET} ${r + QUIET}h1v1h-1z`);
    });
  });
  return parts.join("");
}

/**
 * THE QR CODE of an order's transfer, drawn by the shop's own page from the
 * payload the shop sent: dark modules in ink on the page's ground, with the
 * margin a scanner reads against. It is a picture with a name; every detail it
 * stands for is written beside it as text.
 */
export default function TransferQr({ payload, label }: { payload: string; label: string }) {
  const rows = useMemo(() => qrModules(payload), [payload]);
  const side = rows.length + QUIET * 2;
  return (
    <svg data-testid="transfer-qr" role="img" aria-label={label} viewBox={`0 0 ${side} ${side}`} shapeRendering="crispEdges"
      className="aspect-square w-full max-w-64 bg-ground text-ink">
      <path d={pathOf(rows)} fill="currentColor" />
    </svg>
  );
}
