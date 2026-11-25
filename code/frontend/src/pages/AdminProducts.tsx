import { useContent } from "../editable";
import { useCollection } from "../data";
import type { Category } from "../lib/shop";
import ListPage from "../components/admin/ListPage";
import { columnsFromFields, fieldsOf } from "../components/admin/fields";
import { LIST_COLUMNS, useAdminNav } from "./admin-nav";

/** Every shoe, as a list to search, filter, add to and edit; the form comes from src/data/schema.json. */
export default function AdminProducts() {
  const nav = useAdminNav();
  const keys = LIST_COLUMNS.products;
  const cats = useCollection<Category>("categories", { pageSize: 50 }).items;
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
        { key: "category", label: useContent("pages.admin.products.category"), options: cats.map((c) => [c.id, c.name] as const) },
      ]}
    />
  );
}
