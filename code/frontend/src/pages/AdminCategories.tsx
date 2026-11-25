import { useContent } from "../editable";
import ListPage from "../components/admin/ListPage";
import { useAdminNav } from "./admin-nav";

/** Who the shoes are for — men, women, kids — with the picture each shows on the home page. */
export default function AdminCategories() {
  const nav = useAdminNav();
  return <ListPage collection="categories" title={useContent("pages.admin.categories.title")} nav={nav} search={false} />;
}
