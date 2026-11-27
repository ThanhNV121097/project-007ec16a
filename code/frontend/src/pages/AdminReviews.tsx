import { useContent } from "../editable";
import { useCollection } from "../data";
import type { Product } from "../lib/shop";
import ListPage from "../components/admin/ListPage";
import { columnsFromFields, fieldsOf } from "../components/admin/fields";
import { LIST_COLUMNS, useAdminNav } from "./admin-nav";

/** What customers wrote about each pair; a review is shown or hidden from here. */
export default function AdminReviews() {
  const nav = useAdminNav();
  const keys = LIST_COLUMNS.reviews;
  // The Shoe column names each review's shoe (the dealer's view, drafts included), and a name
  // is looked up among the rows already read (fields.tsx nameIn): the shoes are read here.
  useCollection<Product>("products", { pageSize: 100 });
  return (
    <ListPage
      collection="reviews"
      title={useContent("pages.admin.reviews.title")}
      nav={nav}
      columns={columnsFromFields(fieldsOf("reviews").filter((f) => keys.includes(f.key)), keys.length)}
      searchPlaceholder={useContent("pages.admin.reviews.search")}
      filters={[{ key: "status", label: useContent("pages.admin.reviews.status"), options: [["live", useContent("pages.admin.reviews.live")], ["hidden", useContent("pages.admin.reviews.hidden")]] }]}
    />
  );
}
