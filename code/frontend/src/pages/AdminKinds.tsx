import { T, useContent } from "../editable";
import ListPage from "../components/admin/ListPage";
import { useAdminNav } from "./admin-nav";

/**
 * The kinds the shop sorts its shoes into — men, women, kids — with the
 * picture each shows on the home page. A short list, edited in a dialog;
 * routed at /admin/categories.
 */
export default function AdminKinds() {
  const nav = useAdminNav();
  return (
    <ListPage collection="categories" title={useContent("pages.admin.categories.title")} nav={nav} searchPlaceholder={useContent("pages.admin.categories.search")}>
      <T k="pages.admin.categories.note" as="p" className="max-w-[70ch] text-sm text-ink-soft" />
    </ListPage>
  );
}
