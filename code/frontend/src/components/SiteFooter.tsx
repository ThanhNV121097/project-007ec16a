import { Link } from "react-router-dom";
import { T, useList } from "../editable";
import Logo from "./Logo";
import PhotoCredits from "./PhotoCredits";
import Rule from "./Rule";

type NavLinkItem = { label: string; href: string };

function Column({ heading, list, items }: { heading: string; list: string; items: readonly NavLinkItem[] }) {
  return (
    <nav aria-label={heading} className="flex flex-col gap-2.5 text-sm">
      <T k={heading} as="p" className="mb-1 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft" />
      {items.map((l, i) => (
        <Link key={l.href} to={l.href} className="self-start text-ink underline-offset-4 hover:underline">
          <T k={`${list}.${i}.label`} />
        </Link>
      ))}
    </nav>
  );
}

/**
 * THE FOOTER: a hairline, the wordmark with one line about the shop, three
 * short columns of links, then a second hairline with the line that says
 * this shop is independent of adidas and the stock photographs' credits.
 */
export default function SiteFooter() {
  const shop = useList<NavLinkItem>("footer.shop");
  const help = useList<NavLinkItem>("footer.help");
  const about = useList<NavLinkItem>("footer.about");
  return (
    <footer className="mx-auto mt-16 max-w-page px-[var(--gutter)] pb-[var(--dock-space)] lg:pb-10">
      <Rule />
      <div className="grid grid-cols-2 gap-10 py-12 md:grid-cols-12">
        <div className="col-span-2 md:col-span-5">
          <Logo />
          <T k="footer.line" as="p" className="mt-4 max-w-[40ch] text-sm leading-relaxed text-ink-soft" />
        </div>
        <div className="md:col-span-2"><Column heading="footer.shop_heading" list="footer.shop" items={shop} /></div>
        <div className="md:col-span-3"><Column heading="footer.help_heading" list="footer.help" items={help} /></div>
        <div className="md:col-span-2"><Column heading="footer.about_heading" list="footer.about" items={about} /></div>
      </div>
      <Rule />
      <div className="flex flex-col gap-3 pt-6 text-xs text-ink-soft md:flex-row md:items-start md:justify-between md:gap-10">
        <T k="footer.independent" as="p" className="max-w-[60ch]" />
        <PhotoCredits className="text-xs text-ink-soft md:max-w-[50ch] md:text-right" />
      </div>
    </footer>
  );
}
