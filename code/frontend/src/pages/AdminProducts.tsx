import { useContent } from "../editable";
import ListPage from "../components/admin/ListPage";
import { columnsFromFields, fieldsOf } from "../components/admin/fields";
import { LIST_COLUMNS, useAdminNav } from "./admin-nav";

/** Every shoe, as a list to search, filter, add to and edit; the form comes from src/data/schema.json. */
export default function AdminProducts() {
  const nav = useAdminNav();
  const keys = LIST_COLUMNS.products;
  return (
    <ListPage
      collection="products"
      title={useContent("pages.admin.products.title")}
      nav={nav}
      columns={columnsFromFields(fieldsOf("products").filter((f) => keys.includes(f.key)), keys.length)}
      rowHref={(r) => `/admin/products/${r.id}`}
      searchPlaceholder={useContent("pages.admin.products.search")}
      filters={[
        { key: "status", label: useContent("pages.admin.products.status"), options: [["live", useContent("pages.admin.products.live")], ["draft", useContent("pages.admin.products.draft")]] },
        { key: "category", label: useContent("pages.admin.products.category"), options: [["women", "Women"], ["men", "Men"], ["kids", "Kids"]] },
      ]}
    />
  );
}
