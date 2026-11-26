import { type ReactNode } from "react";
import Rule from "../Rule";

/**
 * One block of My account, drawn as the design draws a band: a hairline, the
 * heading held between two hairlines, then the content. The section is named
 * by its heading, so each block is a landmark a screen reader can jump to.
 */
export default function AccountBlock({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <>
      <Rule />
      <div className="py-7"><h2 id={`${id}-title`} className="text-lg text-ink">{title}</h2></div>
      <Rule />
      <section aria-labelledby={`${id}-title`} className="min-w-0 py-10">{children}</section>
    </>
  );
}
