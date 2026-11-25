import { useContent } from "../editable";
import { useRecord } from "../data";
import { fill, type ShopTerms } from "../lib/shop";
import Icon, { type IconName } from "./Icon";

/**
 * What the shop promises, in four cells on a hairline: delivery, the size
 * exchange, cash on delivery and genuine pairs. The numbers in it are the
 * shop's own terms (src/data/shop.json), so changing a term changes every
 * place it is said.
 */
export default function PromiseRow({ compact = false }: { compact?: boolean }) {
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const values = {
    days: terms?.standard_days ?? "",
    n: terms?.exchange_days ?? "",
    refund: terms?.refund_days ?? "",
  };
  const cells: { icon: IconName; title: string; line: string }[] = [
    { icon: "shipping", title: useContent("promises.delivery_title"), line: fill(useContent("promises.delivery_line"), values) },
    { icon: "returns", title: fill(useContent("promises.exchange_title"), values), line: fill(useContent("promises.exchange_line"), values) },
    { icon: "cash", title: useContent("promises.cod_title"), line: useContent("promises.cod_line") },
    { icon: "quality", title: useContent("promises.genuine_title"), line: useContent("promises.genuine_line") },
  ];
  return (
    <ul className={`grid border-t border-line ${compact ? "grid-cols-1 sm:grid-cols-2" : "grid-cols-2 lg:grid-cols-4"}`}>
      {cells.map((c) => (
        <li key={c.icon} className="flex gap-3 border-b border-line py-5 pr-4">
          <Icon name={c.icon} size={20} className="mt-0.5 shrink-0 text-ink" />
          <div className="min-w-0">
            <p className="text-sm text-ink">{c.title}</p>
            <p className="mt-1 text-sm leading-relaxed text-ink-soft">{c.line}</p>
          </div>
        </li>
      ))}
    </ul>
  );
}
