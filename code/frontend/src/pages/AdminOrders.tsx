import { useContent } from "../editable";
import ListPage from "../components/admin/ListPage";
import { columnsFromFields, fieldsOf } from "../components/admin/fields";
import { formatDateTime } from "../lib/format";
import { LIST_COLUMNS, useAdminNav } from "./admin-nav";

/** Every order, newest first: searchable, filtered by status and payment, each opening the order. */
export default function AdminOrders() {
  const nav = useAdminNav();
  const keys = LIST_COLUMNS.orders;
  const columns = columnsFromFields(fieldsOf("orders").filter((f) => keys.includes(f.key)), keys.length)
    .map((c) => (c.key === "placed_at" ? { ...c, render: (r: { placed_at?: unknown }) => formatDateTime(String(r.placed_at ?? "")) } : c));
  return (
    <ListPage
      collection="orders"
      title={useContent("pages.admin.orders.title")}
      nav={nav}
      add={false}
      columns={columns}
      rowHref={(r) => `/admin/orders/${r.id}`}
      searchPlaceholder={useContent("pages.admin.orders.search")}
      filters={[
        { key: "status", label: useContent("pages.admin.orders.status"), options: [
          ["placed", useContent("orders.steps.placed")],
          ["confirmed", useContent("orders.steps.confirmed")],
          ["packed", useContent("orders.steps.packed")],
          ["shipped", useContent("orders.steps.shipped")],
          ["delivered", useContent("orders.steps.delivered")],
          ["cancelled", useContent("pages.admin.orders.cancelled")],
        ] },
        { key: "payment", label: useContent("pages.admin.orders.payment"), options: [
          ["cod", "COD"], ["bank", "VietQR"], ["vnpay", "VNPay"], ["momo", "MoMo"],
        ] },
      ]}
    />
  );
}
