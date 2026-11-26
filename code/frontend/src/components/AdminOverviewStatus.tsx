import { useContent } from "../editable";
import { Button } from "./admin/index";

/**
 * What stands in place of the figures and panels while the overview is not
 * there: the wait (announced politely) or the failure (announced, with a way
 * to ask again). The title and the menu stay; no figure is drawn — a 0 would
 * be a figure the shop never gave.
 */
export default function AdminOverviewStatus({ status, onRetry }: {
  status: "loading" | "error";
  onRetry: () => void;
}) {
  const loading = useContent("pages.admin.overview.loading");
  const error = useContent("pages.admin.overview.error");
  const retry = useContent("pages.admin.overview.retry");

  if (status === "loading") {
    return <p role="status" className="py-10 text-center text-sm text-ink-soft">{loading}</p>;
  }
  return (
    <div role="alert" className="flex flex-col items-center gap-3 py-10 text-center text-sm text-ink">
      <p>{error}</p>
      <Button onClick={onRetry} className="min-h-11">{retry}</Button>
    </div>
  );
}
