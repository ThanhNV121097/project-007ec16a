import { Link } from "react-router-dom";
import { T, useContent } from "../editable";
import OverviewPage from "../components/admin/OverviewPage";
import AdminBars from "../components/AdminBars";
import ProductImage from "../components/ProductImage";
import { useCollection } from "../data";
import { formatDateTime, formatPrice } from "../lib/format";
import { fill, type Line, type Order, type Product } from "../lib/shop";
import { useAdminNav } from "./admin-nav";

function dayKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

/** The shoes down to their last pairs, as a short list of links: the picture, the name, the sizes, the count. */
function LowList({ rows, sizesText }: { rows: Product[]; sizesText: string }) {
  return (
    <ul className="divide-y divide-line">
      {rows.map((p) => (
        <li key={p.id}>
          <Link to={`/admin/products/${p.id}`} className="flex items-center gap-3 py-2.5 hover:bg-wash">
            <span className="w-10 shrink-0"><ProductImage src={p.image} alt={p.name} aspect="1/1" /></span>
            <span className="min-w-0 flex-1 text-sm">
              <span className="block font-medium text-ink">{p.name}</span>
              <span className="block text-ink-soft">{fill(sizesText, { sizes: (p.sizes_low ?? []).join(", ") })}</span>
            </span>
            <span className="shrink-0 text-sm tabular-nums text-ink">{p.stock ?? 0}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

/** The latest orders, each a sentence a person can read at a glance. */
function RecentList({ rows, products, statuses }: { rows: Order[]; products: Product[]; statuses: Record<string, string> }) {
  return (
    <ul className="divide-y divide-line">
      {rows.map((o) => (
        <li key={o.id}>
          <Link to={`/admin/orders/${o.id}`} className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-2.5 text-sm hover:bg-wash">
            <span className="min-w-0">
              <span className="font-medium tabular-nums text-ink">{o.number}</span>{" · "}
              <span className="text-ink">{o.customer}</span>{" · "}
              <span className="text-ink-soft">{products.find((p) => p.id === o.product)?.name} · EU {o.size}</span>
            </span>
            <span className="shrink-0 text-ink-soft">
              {statuses[o.status] ?? o.status} · <span className="tabular-nums">{formatDateTime(o.placed_at)}</span> · <span className="tabular-nums text-ink">{formatPrice(o.total)}</span>
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

/**
 * THE DEALER'S FIRST SCREEN: four figures — this month's sales, what is
 * waiting to be packed, pairs on the shelf, shoes running out — each opening
 * the list it counts; then sales over the last fourteen days, pairs sold by
 * line, the latest orders and the shoes running low.
 */
export default function AdminOverview() {
  const nav = useAdminNav();
  const orders = useCollection<Order>("orders", { pageSize: 1000, sort: "-placed_at" }).items;
  const products = useCollection<Product>("products", { pageSize: 1000 }).items;
  const lines = useCollection<Line>("lines", { pageSize: 50 }).items;
  const ordersText = useContent("pages.admin.overview.orders_note");
  const sizesText = useContent("pages.admin.overview.low_sizes");
  const statuses = { ...useContent<Record<string, string>>("orders.steps"), cancelled: useContent("pages.admin.orders.cancelled") };

  const live = orders.filter((o) => o.status !== "cancelled");
  const latest = live.reduce((d, o) => (o.placed_at > d ? o.placed_at : d), "");
  const end = latest ? new Date(latest) : new Date();
  const month = live.filter((o) => {
    const d = new Date(o.placed_at);
    return d.getFullYear() === end.getFullYear() && d.getMonth() === end.getMonth();
  });
  const sales = month.reduce((s, o) => s + o.total, 0);
  const waiting = live.filter((o) => o.status === "placed" || o.status === "confirmed").length;
  const pairs = products.reduce((s, p) => s + (p.stock ?? 0), 0);
  const low = products.filter((p) => (p.sizes_low?.length ?? 0) > 0 && (p.stock ?? 0) < 15).sort((a, b) => (a.stock ?? 0) - (b.stock ?? 0));

  const days = Array.from({ length: 14 }, (_, i) => {
    const d = new Date(end);
    d.setDate(d.getDate() - (13 - i));
    const key = dayKey(d);
    const value = live.filter((o) => o.placed_at.slice(0, 10) === key).reduce((s, o) => s + o.total, 0);
    return { label: String(d.getDate()), value, display: formatPrice(value) };
  });
  const byLine = lines.map((l) => {
    const sold = live.filter((o) => products.find((p) => p.id === o.product)?.line === l.id).reduce((s, o) => s + o.qty, 0);
    return { label: l.name, value: sold, display: String(sold) };
  }).sort((a, b) => b.value - a.value);

  return (
    <OverviewPage
      title={useContent("pages.admin.overview.title")}
      nav={nav}
      stats={[
        { label: useContent("pages.admin.overview.sales"), value: formatPrice(sales), icon: "cash", to: "/admin/orders",
          note: fill(ordersText, { n: month.length }) },
        { label: useContent("pages.admin.overview.to_pack"), value: waiting, icon: "package", to: "/admin/orders?status=confirmed",
          note: useContent("pages.admin.overview.to_pack_note") },
        { label: useContent("pages.admin.overview.stock"), value: pairs, icon: "tag", to: "/admin/products",
          note: fill(useContent("pages.admin.overview.stock_note"), { n: products.length }) },
        { label: useContent("pages.admin.overview.low"), value: low.length, icon: "alert", to: "/admin/products",
          note: useContent("pages.admin.overview.low_note") },
      ]}
      panels={[
        { title: useContent("pages.admin.overview.chart_title"), content: <AdminBars rows={days} caption={useContent("pages.admin.overview.chart_title")} /> },
        { title: useContent("pages.admin.overview.lines_title"), content: <AdminBars kind="rows" rows={byLine} caption={useContent("pages.admin.overview.lines_title")} /> },
        { title: useContent("pages.admin.overview.recent_title"), content: <RecentList rows={orders.slice(0, 5)} products={products} statuses={statuses} /> },
        { title: useContent("pages.admin.overview.low_title"),
          content: low.length > 0 ? <LowList rows={low} sizesText={sizesText} /> : <T k="pages.admin.overview.low_none" as="p" className="text-sm text-ink-soft" /> },
      ]}
    />
  );
}
