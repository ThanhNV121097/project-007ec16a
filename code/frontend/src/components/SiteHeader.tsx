import { useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { T, useContent, useList } from "../editable";
import MenuToggle from "./MenuToggle";
import MobileMenu from "./MobileMenu";
import Rule from "./Rule";

/**
 * THE HEADER: the wordmark at the left, plain links at the right, and a
 * hairline under both. No bar, no fill, no button — on this site even the
 * header is type on white.
 */
export default function SiteHeader() {
  const name = useContent("site.name");
  const links = useList<{ label: string; href: string }>("nav.links");
  const [open, setOpen] = useState(false);
  return (
    <header className="mx-auto max-w-page px-[var(--gutter)]">
      <div className="flex h-[var(--header-h)] items-center justify-between">
        <Link to="/" aria-label={name} className="font-display text-lg tracking-tight text-ink">
          <T k="site.name" />
        </Link>
        <nav className="hidden items-center gap-7 text-sm lg:flex" aria-label="Main">
          {links.map((l, i) => (
            <NavLink key={l.href} to={l.href}
              className={({ isActive }) => `text-ink underline-offset-8 hover:underline ${isActive ? "underline" : ""}`}>
              <T k={`nav.links.${i}.label`} />
            </NavLink>
          ))}
        </nav>
        <MenuToggle open={open} onToggle={() => setOpen((o) => !o)} className="lg:hidden" />
      </div>
      <Rule />
      <MobileMenu open={open} onClose={() => setOpen(false)}>
        <nav className="flex flex-col gap-6 text-3xl" aria-label="Main">
          {links.map((l, i) => (
            <Link key={l.href} to={l.href} onClick={() => setOpen(false)} className="text-ink">
              <T k={`nav.links.${i}.label`} />
            </Link>
          ))}
        </nav>
      </MobileMenu>
    </header>
  );
}
