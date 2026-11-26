import AdminOverviewReady from "../components/AdminOverviewReady";
import AdminOverviewStatus from "../components/AdminOverviewStatus";
import OverviewPage from "../components/admin/OverviewPage";
import { useContent } from "../editable";
import { useOverview } from "../lib/overview";
import { useAdminNav } from "./admin-nav";

/**
 * THE DEALER'S FIRST SCREEN. One read of the shop's overview feeds all of it:
 * while the answer is awaited, or when it does not come, the title and the
 * menu stay and a line under the title says so; once it is here the figures
 * and panels are drawn from it.
 */
export default function AdminOverview() {
  const nav = useAdminNav();
  const title = useContent("pages.admin.overview.title");
  const overview = useOverview();

  if (overview.status === "ready") return <AdminOverviewReady data={overview.data} title={title} nav={nav} />;
  return (
    <OverviewPage title={title} nav={nav} stats={[]}>
      <AdminOverviewStatus status={overview.status} onRetry={overview.retry} />
    </OverviewPage>
  );
}
