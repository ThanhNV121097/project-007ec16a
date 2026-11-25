import { useState } from "react";
import { Navigate } from "react-router-dom";
import DashboardLayout from "../components/DashboardLayout";
import StaffAccounts, { type StaffChoice } from "../components/StaffAccounts";
import StaffDialogs, { type StaffDialog } from "../components/StaffDialogs";
import { Button, Notice } from "../components/admin/index";
import { useContent } from "../editable";
import { useRole } from "../lib/session";
import { useStaff, type StaffAccount } from "../lib/staff";
import { useAdminNav } from "./admin-nav";

interface Said {
  tone: "done" | "alert";
  text: string;
}

/** The team's accounts: the admin adds, resets, switches off and removes. */
function StaffPage() {
  const nav = useAdminNav();
  const staff = useStaff();
  const [dialog, setDialog] = useState<StaffDialog | null>(null);
  const [said, setSaid] = useState<Said | null>(null);
  const savedText = useContent("pages.admin.staff.dialog.reset.saved");
  const couldNotSave = useContent("pages.admin.staff.couldNotSave");

  // Switching on asks nothing, so there is no dialog to say it failed in:
  // the sentence goes above the table.
  const switchOn = (account: StaffAccount) =>
    staff.update(account.id, { is_active: true }).then((result) => {
      if (!result.ok) setSaid({ tone: "alert", text: result.kind === "refused" ? result.message : couldNotSave });
    });

  const choose = (choice: StaffChoice, account: StaffAccount) => {
    setSaid(null);
    if (choice === "on") void switchOn(account);
    else setDialog({ kind: choice, account });
  };

  const add = (
    <Button kind="primary" icon="plus" onClick={() => { setSaid(null); setDialog({ kind: "add" }); }}>
      {useContent("pages.admin.staff.add")}
    </Button>
  );

  return (
    <DashboardLayout title={useContent("pages.admin.staff.title")} nav={nav} actions={staff.status === "ready" ? add : null}>
      {said && <Notice tone={said.tone} onClose={() => setSaid(null)}>{said.text}</Notice>}
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
      <StaffAccounts items={staff.items} status={staff.status} onRetry={staff.retry} onChoose={choose} />
=======
      <StaffAccounts items={staff.items} total={staff.total} status={staff.status} onRetry={staff.retry} onChoose={choose} />
>>>>>>> refs/heads/main
      <StaffDialogs
        dialog={dialog}
        actions={staff}
        onClose={() => setDialog(null)}
        onSaved={() => setSaid({ tone: "done", text: savedText })}
      />
    </DashboardLayout>
  );
}

/**
 * Staff manage nothing here: a staff session is sent to the overview before
 * anything is drawn or asked for, so the list is never fetched for them. A
 * customer and a visitor never reach this far: the /admin gate answers them.
 */
export default function AdminStaff() {
  if (useRole() === "staff") return <Navigate to="/admin" replace />;
  return <StaffPage />;
}
