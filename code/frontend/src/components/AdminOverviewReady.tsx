import AdminBars from "./AdminBars";
import AdminLowStock from "./AdminLowStock";
import AdminRecentOrders from "./AdminRecentOrders";
import OverviewPage from "./admin/OverviewPage";
import type { NavGroup } from "./DashboardLayout";
import { T, useContent } from "../editable";
import { formatPrice } from "../lib/format";
import { dayOfMonth, type Overview } from "../lib/overview";
import { fill } from "../lib/shop";

/**
 * THE DEALER'S FIRST SCREEN, once the shop has answered: four figures — this
 * month's sales, what is waiting to be packed, pairs on the shelf, shoes
 * running out — each opening the list it counts; then "Today", sales over the
 * last fourteen days, pairs sold by line, the latest orders and the shoes
 * running low. Everything comes from the one answer it is given, so the
 * sentence's counts are always the figures' own.
 */
export default function AdminOverviewReady({ data, title, nav }: {
  data: Overview;
  title: string;
  nav: readonly NavGroup[];
}) {
  const chartTitle = useContent("pages.admin.overview.chart_title");
  const linesTitle = useContent("pages.admin.overview.lines_title");
  const sizesText = useContent("pages.admin.overview.low_sizes");
  const sales = formatPrice(data.month.sales);
  const low = data.running_low.length;

  const days = data.sales_by_day.map((d) => ({
    label: dayOfMonth(d.date),
    value: d.sales,
    display: formatPrice(d.sales),
  }));
  const byLine = data.pairs_by_line.map((l) => ({ label: l.name, value: l.pairs, display: String(l.pairs) }));

  return (
    <OverviewPage
      title={title}
      nav={nav}
      stats={[
        { label: useContent("pages.admin.overview.sales"), value: sales, icon: "cash", to: "/admin/orders",
          note: fill(useContent("pages.admin.overview.orders_note"), { n: data.month.orders }) },
        { label: useContent("pages.admin.overview.to_pack"), value: data.waiting_orders, icon: "package", to: "/admin/orders?status=confirmed",
          note: useContent("pages.admin.overview.to_pack_note") },
        { label: useContent("pages.admin.overview.stock"), value: data.pairs_on_shelf, icon: "tag", to: "/admin/products",
          note: fill(useContent("pages.admin.overview.stock_note"), { n: data.shoes }) },
        { label: useContent("pages.admin.overview.low"), value: low, icon: "alert", to: "/admin/products",
          note: useContent("pages.admin.overview.low_note") },
      ]}
      panels={[
        { title: useContent("pages.admin.overview.today_title"), content: (
          <p className="text-sm text-ink">
            {fill(useContent("pages.admin.overview.today_text"), { waiting: data.waiting_orders, low, sales, n: data.month.orders })}
          </p>
        ) },
        { title: chartTitle, content: <AdminBars rows={days} caption={chartTitle} /> },
        { title: linesTitle, content: <AdminBars kind="rows" rows={byLine} caption={linesTitle} /> },
        { title: useContent("pages.admin.overview.recent_title"), content: <AdminRecentOrders rows={data.latest_orders} /> },
        { title: useContent("pages.admin.overview.low_title"),
          content: low > 0
            ? <AdminLowStock rows={data.running_low} sizesText={sizesText} />
            : <T k="pages.admin.overview.low_none" as="p" className="text-sm text-ink-soft" /> },
      ]}
    />
  );
}
