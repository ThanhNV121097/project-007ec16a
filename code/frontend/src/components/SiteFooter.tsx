import { Link } from "react-router-dom";
import { T, useList } from "../editable";
import PhotoCredits from "./PhotoCredits";
import Rule from "./Rule";

/**
 * THE FOOTER: a hairline, then the same links as the header and one line
 * about the shop. Small, and on white like everything else. The stock
 * photographs' credits stay at its foot — their licences ask for them.
 */
export default function SiteFooter() {
  const links = useList<{ label: string; href: string }>("nav.links");
  return (
    <footer className="mx-auto max-w-page px-[var(--gutter)] pb-10">
      <Rule />
      <div className="flex flex-col gap-6 py-10 text-sm sm:flex-row sm:items-start sm:justify-between">
        <div className="max-w-[36ch]">
          <T k="site.name" as="p" className="font-display text-lg tracking-tight text-ink" />
          <T k="footer.line" as="p" className="mt-2 text-ink-soft" />
        </div>
        <nav className="flex flex-wrap gap-x-7 gap-y-2" aria-label="Footer">
          {links.map((l, i) => (
            <Link key={l.href} to={l.href} className="text-ink underline-offset-8 hover:underline">
              <T k={`nav.links.${i}.label`} />
            </Link>
          ))}
          <Link to="/admin" className="text-ink-soft underline-offset-8 hover:underline"><T k="footer.admin" /></Link>
        </nav>
      </div>
      <PhotoCredits className="text-xs text-ink-soft" />
    </footer>
  );
}
