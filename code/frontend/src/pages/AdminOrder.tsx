import { useParams } from "react-router-dom";
import { useContent } from "../editable";
import RecordPage from "../components/admin/RecordPage";
import { useAdminNav } from "./admin-nav";

/** One order, open: its status, the pair, delivery and the customer, on the tabs the schema gives them. */
export default function AdminOrder() {
  const { id } = useParams();
  const nav = useAdminNav();
  return <RecordPage collection="orders" id={id} nav={nav} backTo="/admin/orders" title={useContent("pages.admin.orders.title")} />;
}
