import { useContent } from "../editable";
import { useCollection } from "../data";
import type { OrderStatus, PaymentMethod, Product } from "../lib/shop";
import ListPage from "../components/admin/ListPage";
import { Badge } from "../components/admin/index";
import { columnsFromFields, fieldsOf, toneOf, type FieldSpec } from "../components/admin/fields";
import { formatDateTime } from "../lib/format";
import { EDITABLE, useStatusLabels } from "../lib/dealer-handles-orders-status-courier-payment";
import type { Column } from "../components/DataTable";
import type { Row } from "../data";
import { LIST_COLUMNS, useAdminNav } from "./admin-nav";

/** Every order, newest first: searchable, filtered by status and payment, each opening the order. */
export default function AdminOrders() {
  const nav = useAdminNav();
  const keys = LIST_COLUMNS.orders;
  const labels = useStatusLabels();
  const methods = useCollection<PaymentMethod>("payments", { pageSize: 20 }).items;
  const shoes = new Map(useCollection<Product>("products", { pageSize: 100 }).items.map((p) => [p.id, p.name]));
  const columns = columnsFromFields(fieldsOf("orders").filter((f) => keys.includes(f.key)), keys.length)
    .map((c): Column<Row> => {
      // The shoe is its name, once: the first pair of an order of several. The shop's list names it
      // (`product_name`); the shoes read here are the fallback while a list from another source has none.
      if (c.key === "product") {
        return { ...c, render: (r: Row) => String(r.product_name ?? "") || (shoes.get(String(r.product)) ?? String(r.product ?? "")) };
      }
      if (c.key === "placed_at") return { ...c, render: (r: Row) => formatDateTime(String(r.placed_at ?? "")) };
      if (c.key === "status") {
        return { ...c, render: (r: Row) => <Badge tone={toneOf(String(r.status))}>{labels[r.status as OrderStatus] ?? String(r.status)}</Badge> };
      }
      return c;
    });
  // The row menu's Edit offers what the dealer can change and nothing the shop refuses.
  const fields: FieldSpec[] = fieldsOf("orders")
    .filter((f) => (EDITABLE as readonly string[]).includes(f.key))
    .map((f) => (f.key === "status" ? { ...f, options: Object.entries(labels) } : f));
  return (
    <ListPage
      collection="orders"
      title={useContent("pages.admin.orders.title")}
      nav={nav}
      add={false}
      columns={columns}
      fields={fields}
      rowHref={(r) => `/admin/orders/${r.id}`}
      searchPlaceholder={useContent("pages.admin.orders.search")}
      filters={[
        { key: "status", label: useContent("pages.admin.orders.status"), options: [
          ["placed", labels.placed],
          ["confirmed", labels.confirmed],
          ["packed", labels.packed],
          ["shipped", labels.shipped],
          ["delivered", labels.delivered],
          ["cancelled", labels.cancelled],
        ] },
        { key: "payment", label: useContent("pages.admin.orders.payment"), options: methods.map((m) => [m.id, m.name] as const) },
      ]}
    />
  );
}
