import { T, useContent } from "../editable";
import ListPage from "../components/admin/ListPage";
import { useAdminNav } from "./admin-nav";

/** The adidas lines the shop sorts its shoes into — a short list, edited in a dialog. */
export default function AdminLines() {
  const nav = useAdminNav();
  return (
    <ListPage collection="lines" title={useContent("pages.admin.lines.title")} nav={nav} searchPlaceholder={useContent("pages.admin.lines.search")}>
      <T k="pages.admin.lines.note" as="p" className="max-w-[70ch] text-sm text-ink-soft" />
    </ListPage>
  );
}
