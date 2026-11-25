import { useContent } from "../editable";
import ListPage from "../components/admin/ListPage";
import { columnsFromFields, fieldsOf } from "../components/admin/fields";
import { LIST_COLUMNS, useAdminNav } from "./admin-nav";

/** What customers wrote about each pair; a review is shown or hidden from here. */
export default function AdminReviews() {
  const nav = useAdminNav();
  const keys = LIST_COLUMNS.reviews;
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
