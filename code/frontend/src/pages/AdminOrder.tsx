import { useParams } from "react-router-dom";
import { useContent } from "../editable";
import RecordPage from "../components/admin/RecordPage";
import AdminOrderPayment from "../components/AdminOrderPayment";
import ProductImage from "../components/ProductImage";
import { useRecord } from "../data";
import { formatDateTime, formatPrice } from "../lib/format";
import { fill, type Order, type Product } from "../lib/shop";
import { useAdminNav } from "./admin-nav";

/**
 * One order, open: a sentence and the pair's picture at the top — who, what,
 * when, how much — then its status, delivery and customer on the tabs the
 * schema gives them.
 */
export default function AdminOrder() {
  const { id } = useParams();
  const nav = useAdminNav();
  const order = useRecord<Order>("orders", id).item;
  const product = useRecord<Product>("products", order?.product).item;
  const summary = useContent("pages.admin.orders.summary");
  return (
    <RecordPage collection="orders" id={id} nav={nav} backTo="/admin/orders" title={useContent("pages.admin.orders.title")}>
      {order && (
        <>
          <div className="flex items-center gap-4 rounded-md border border-line bg-surface p-4">
            <span className="w-16 shrink-0"><ProductImage src={product?.image} alt={product?.name ?? ""} aspect="1/1" /></span>
            <p className="text-sm text-ink">{fill(summary, {
              customer: order.customer, city: order.city ?? "", qty: order.qty, product: product?.name ?? "",
              size: order.size, date: formatDateTime(order.placed_at), total: formatPrice(order.total),
            })}</p>
          </div>
          <AdminOrderPayment id={order.id} way={order.payment} paid={order.paid} status={order.status} />
        </>
      )}
    </RecordPage>
  );
}
