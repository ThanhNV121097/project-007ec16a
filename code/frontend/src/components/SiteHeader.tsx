import { useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { T, useContent, useList } from "../editable";
import { useCart } from "../lib/cart";
import { closeBag, openBag, useBagOpen } from "../lib/bag";
import CartDrawer from "./CartDrawer";
import Icon from "./Icon";
import Logo from "./Logo";
import MenuToggle from "./MenuToggle";
import MobileMenu from "./MobileMenu";
import Rule from "./Rule";

/**
 * THE HEADER: the wordmark at the left, five plain links at the right, then
 * search and the bag as icons — and a hairline under all of it. No bar, no
 * fill. Search opens a line of its own under the links and sends the words
 * to the catalogue; the bag opens the drawer.
 */
export default function SiteHeader() {
  const name = useContent("site.name");
  const links = useList<{ label: string; href: string }>("nav.links");
  const help = useList<{ label: string; href: string }>("footer.help");
  const searchLabel = useContent("nav.search");
  const bagLabel = useContent("nav.bag");
  const placeholder = useContent("nav.search_placeholder");
  const { count } = useCart();
  const bagOpen = useBagOpen();
  const navigate = useNavigate();
  const { pathname, search } = useLocation();
  const here = `${pathname}${search}`;
  const [open, setOpen] = useState(false);
  const [searching, setSearching] = useState(false);
  const [words, setWords] = useState("");

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const q = words.trim();
    setSearching(false);
    setOpen(false);
    navigate(q ? `/shop?q=${encodeURIComponent(q)}` : "/shop");
  };

  return (
    <header className="mx-auto max-w-page px-[var(--gutter)]">
      <div className="flex h-[var(--header-h)] items-center justify-between gap-6">
        <Link to="/" aria-label={name} onClick={() => setOpen(false)}><Logo /></Link>
        <div className="flex items-center gap-4 lg:gap-8">
          <nav className="hidden items-center gap-7 text-sm lg:flex" aria-label="Main">
            {links.map((l, i) => (
              <Link key={l.href} to={l.href} aria-current={here === l.href ? "page" : undefined}
                className={`text-ink underline-offset-8 hover:underline ${here === l.href ? "underline" : ""}`}>
                <T k={`nav.links.${i}.label`} />
              </Link>
            ))}
          </nav>
          <div className="-mr-2 flex items-center">
            <button type="button" aria-label={searchLabel} aria-expanded={searching} onClick={() => setSearching((s) => !s)}
              className="grid size-10 place-items-center text-ink hover:opacity-70">
              <Icon name={searching ? "close" : "search"} />
            </button>
            <button type="button" aria-label={bagLabel} onClick={openBag} data-testid="bag-button"
              className="relative grid size-10 place-items-center text-ink hover:opacity-70">
              <Icon name="bag" />
              {count > 0 && (
                <span data-testid="bag-count"
                  className="absolute right-0.5 top-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-ink px-1 text-[0.625rem] font-medium tabular-nums text-ground">
                  {count}
                </span>
              )}
            </button>
            <MenuToggle open={open} onToggle={() => setOpen((o) => !o)} className="lg:hidden" />
          </div>
        </div>
      </div>
      {searching && (
        <form role="search" onSubmit={submit} className="flex items-center gap-3 border-t border-line py-3">
          <Icon name="search" className="text-ink-soft" />
          <input autoFocus type="search" value={words} onChange={(e) => setWords(e.target.value)} placeholder={placeholder}
            aria-label={searchLabel}
            className="min-w-0 flex-1 appearance-none bg-transparent py-2 text-base text-ink outline-none placeholder:text-ink-soft" />
          <button type="submit" className="rounded-[var(--radius-pill)] border border-ink px-4 py-1.5 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink hover:bg-ink hover:text-ground">
            <T k="nav.search_go" />
          </button>
        </form>
      )}
      <Rule />
      <MobileMenu open={open} onClose={() => setOpen(false)}>
        <nav className="flex flex-col gap-5 text-3xl tracking-tight" aria-label="Main">
          {links.map((l, i) => (
            <Link key={l.href} to={l.href} onClick={() => setOpen(false)} className="text-ink">
              <T k={`nav.links.${i}.label`} />
            </Link>
          ))}
        </nav>
        <Rule />
        <nav className="flex flex-col gap-3 text-base" aria-label="Help">
          {help.map((l, i) => (
            <Link key={l.href} to={l.href} onClick={() => setOpen(false)} className="text-ink-soft hover:text-ink">
              <T k={`footer.help.${i}.label`} />
            </Link>
          ))}
        </nav>
      </MobileMenu>
      <CartDrawer open={bagOpen} onClose={closeBag} checkout="/checkout"
        title={<T k="cart.title" />} empty={<T k="cart.empty" />} pay={<T k="cart.pay" />} totalLabel={<T k="cart.total" />} />
    </header>
  );
}
