import { useNavigate, useParams } from "react-router-dom";
import { useContent } from "../editable";
import DashboardLayout from "../components/DashboardLayout";
import OrderRecord from "../components/OrderRecord";
import OrderState from "../components/OrderState";
import { Breadcrumb } from "../components/admin/index";
import { titleOf } from "../components/admin/fields";
import { useCollection } from "../data";
import { useDealerOrder } from "../lib/dealer-handles-orders-status-courier-payment";
import type { DeliveryOption, PaymentMethod } from "../lib/shop";
import { useAdminNav } from "./admin-nav";

const BACK = "/admin/orders";

/** The name of a record in a short list, or its id while the list is on its way. */
function nameIn(list: readonly { id: string; name: string }[], id: string): string {
  return list.find((r) => r.id === id)?.name ?? id;
}

/**
 * One order, open: its picture and sentence, then its status, delivery and
 * customer on three tabs. The page asks the shop for the order and says what
 * it has: still waiting, could not ask, no such order, or the order itself.
 * "This record is not here" is only ever the shop's own answer.
 */
export default function AdminOrder() {
  const { id } = useParams();
  const navigate = useNavigate();
  const nav = useAdminNav();
  const { load, replace, retry } = useDealerOrder(id);
  const methods = useCollection<PaymentMethod>("payments", { pageSize: 20 }).items;
  const options = useCollection<DeliveryOption>("delivery", { pageSize: 20 }).items;
  const area = useContent("pages.admin.orders.title");
  const order = load.state === "ready" ? load.order : undefined;
  const heading = order ? titleOf({ ...order }) : "";
  return (
    <DashboardLayout title={area} nav={nav}>
      <Breadcrumb trail={[{ label: area, to: BACK }, { label: heading }]} />
      {load.state !== "ready" && <OrderState state={load.state} onRetry={retry} onBack={() => navigate(BACK)} />}
      {order && (
        <OrderRecord order={order} title={heading} backTo={BACK} onSaved={replace}
          paymentName={nameIn(methods, order.payment)} deliveryName={nameIn(options, order.delivery)} />
      )}
    </DashboardLayout>
  );
}
