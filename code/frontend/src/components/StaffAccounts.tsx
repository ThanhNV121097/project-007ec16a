import { useContent } from "../editable";
import type { StaffAccount, StaffStatus } from "../lib/staff";
import DataTable, { type Column } from "./DataTable";
import { Badge, Button, RowMenu } from "./admin/index";

export type StaffChoice = "reset" | "off" | "on" | "remove";

/** The words of the table, read once so the columns and the menu share them. */
function useWords() {
  return {
    name: useContent("pages.admin.staff.columns.name"),
    email: useContent("pages.admin.staff.columns.email"),
    role: useContent("pages.admin.staff.columns.role"),
    status: useContent("pages.admin.staff.columns.status"),
    admin: useContent("pages.admin.staff.roles.admin"),
    staff: useContent("pages.admin.staff.roles.staff"),
    active: useContent("pages.admin.staff.status.active"),
    off: useContent("pages.admin.staff.status.off"),
    reset: useContent("pages.admin.staff.menu.reset"),
    switchOff: useContent("pages.admin.staff.menu.off"),
    switchOn: useContent("pages.admin.staff.menu.on"),
    remove: useContent("pages.admin.staff.menu.remove"),
    title: useContent("pages.admin.staff.title"),
    loading: useContent("pages.admin.staff.loading"),
    error: useContent("pages.admin.staff.error"),
    retry: useContent("pages.admin.staff.retry"),
  };
}

/**
 * The team, as the shop gave it: its own row first (no menu: that account is
 * never changed from here), then one row per staff member, each with a menu
 * of what can be done to them. Three views and no more: waiting for the
 * answer, the answer, and the answer that did not come.
 *
 * Everything it draws arrives as props — the accounts, how many the shop says
 * there are, whether they have arrived, and what to do when one is chosen — so
 * the page owns the data.
 */
export default function StaffAccounts({ items, total, status, onRetry, onChoose }: {
  items: readonly StaffAccount[];
  total: number;
  status: StaffStatus;
  onRetry: () => void;
  onChoose: (choice: StaffChoice, account: StaffAccount) => void;
}) {
  const w = useWords();

  if (status === "loading") {
    return <p role="status" className="py-10 text-center text-sm text-ink-soft">{w.loading}</p>;
  }
  if (status === "error") {
    return (
      <div role="alert" className="flex flex-col items-center gap-3 py-10 text-center text-sm text-ink">
        <p>{w.error}</p>
        <Button onClick={onRetry}>{w.retry}</Button>
      </div>
    );
  }

  const columns: Column<StaffAccount>[] = [
    { key: "name", label: w.name, render: (r) => <span className="font-medium">{r.name}</span> },
    { key: "email", label: w.email },
    { key: "role", label: w.role, render: (r) => (r.role === "admin" ? w.admin : w.staff) },
    {
      key: "status",
      label: w.status,
      render: (r) => <Badge tone={r.is_active ? "good" : "off"}>{r.is_active ? w.active : w.off}</Badge>,
    },
  ];

  return (
    <DataTable<StaffAccount>
      columns={columns}
      rows={items}
      rowKey={(r) => r.id}
      total={total}
      page={1}
      pages={1}
      size={100}
      onPage={() => undefined}
      caption={w.title}
      export={false}
      actions={(row) =>
        row.role === "admin" ? null : (
          <RowMenu
            items={[
              { label: w.reset, onClick: () => onChoose("reset", row) },
              row.is_active
                ? { label: w.switchOff, onClick: () => onChoose("off", row) }
                : { label: w.switchOn, onClick: () => onChoose("on", row) },
              { label: w.remove, onClick: () => onChoose("remove", row), danger: true },
            ]}
          />
        )
      }
    />
  );
}
