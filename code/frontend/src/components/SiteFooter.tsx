import { Link } from "react-router-dom";
import { T, useList } from "../editable";
import Logo from "./Logo";
import PhotoCredits from "./PhotoCredits";

type NavLinkItem = { label: string; href: string };

function Column({ heading, list, items }: { heading: string; list: string; items: readonly NavLinkItem[] }) {
  return (
    <nav aria-label={heading} className="flex flex-col gap-2.5 text-sm">
      <T k={heading} as="p" className="mb-1 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-accent-ink/60" />
      {items.map((l, i) => (
        <Link key={l.href} to={l.href} className="self-start text-accent-ink underline-offset-4 hover:underline">
          <T k={`${list}.${i}.label`} />
        </Link>
      ))}
    </nav>
  );
}

/**
 * THE FOOTER: the one surface on the site painted in the ink — a black band
 * the full width of the screen, so every page ends on the template's own
 * colour. Inside it the same hairline system, drawn in the ground's colour:
 * the wordmark and a line about the shop, three short columns of links, then
 * the line that says this shop is independent of adidas and the stock
 * photographs' credits.
 */
export default function SiteFooter() {
  const shop = useList<NavLinkItem>("footer.shop");
  const help = useList<NavLinkItem>("footer.help");
  const about = useList<NavLinkItem>("footer.about");
  return (
    <footer className="mt-16 bg-accent text-accent-ink">
      <div className="mx-auto max-w-page px-[var(--gutter)] pb-[var(--dock-space)] pt-4 lg:pb-12">
        <div className="grid grid-cols-2 gap-10 py-12 md:grid-cols-3 lg:grid-cols-12">
          <div className="col-span-2 md:col-span-3 lg:col-span-5">
            <Logo />
            <T k="footer.line" as="p" className="mt-4 max-w-[40ch] text-sm leading-relaxed text-accent-ink/70" />
          </div>
          <div className="lg:col-span-2"><Column heading="footer.shop_heading" list="footer.shop" items={shop} /></div>
          <div className="lg:col-span-3"><Column heading="footer.help_heading" list="footer.help" items={help} /></div>
          <div className="lg:col-span-2"><Column heading="footer.about_heading" list="footer.about" items={about} /></div>
        </div>
        <hr className="h-px w-full border-0 bg-accent-ink/15" />
        <div className="flex flex-col gap-3 pt-6 text-[0.8125rem] leading-relaxed text-accent-ink/60 md:flex-row md:items-start md:justify-between md:gap-10">
          <T k="footer.independent" as="p" className="max-w-[60ch]" />
          <PhotoCredits className="text-[0.8125rem] leading-relaxed text-accent-ink/70 md:max-w-[50ch] md:text-right" />
        </div>
      </div>
    </footer>
  );
}
