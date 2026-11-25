import { useContent } from "../editable";
import { useCollection } from "../data";
import type { NavGroup } from "../components/DashboardLayout";

/**
 * The management area's menu, once, for every page of it: the overview,
 * the orders (with how many are waiting to be packed), and the catalogue.
 * Its words come from content.json so the owner can rename them.
 */
export function useAdminNav(): NavGroup[] {
  const waiting = useCollection("orders", { where: { status: ["placed", "confirmed"] }, pageSize: 1 }).total;
  return [
    { items: [{ to: "/admin", label: useContent("pages.admin.nav.overview") }] },
    {
      heading: useContent("pages.admin.nav.sales"),
      items: [{ to: "/admin/orders", label: useContent("pages.admin.nav.orders"), badge: waiting || undefined }],
    },
    {
      heading: useContent("pages.admin.nav.catalogue"),
      items: [
        { to: "/admin/products", label: useContent("pages.admin.nav.products") },
        { to: "/admin/lines", label: useContent("pages.admin.nav.lines") },
        { to: "/admin/categories", label: useContent("pages.admin.nav.categories") },
        { to: "/admin/reviews", label: useContent("pages.admin.nav.reviews") },
      ],
    },
  ];
}

/** The columns of a list, picked by key from the schema's own fields. */
export const LIST_COLUMNS: Record<string, readonly string[]> = {
  orders: ["number", "placed_at", "status", "product", "total", "payment", "customer"],
  products: ["image", "name", "line", "category", "price", "status", "stock"],
  reviews: ["product", "name", "rating", "date", "status"],
};
