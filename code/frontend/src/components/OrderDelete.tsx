import { useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "./ui/alert-dialog";
import { deleteOrder, type DeleteResult } from "../lib/dealer-handles-orders-status-courier-payment";

/** What the open question says when the shop did not delete. */
function refusal(result: Exclude<DeleteResult, { ok: true }>): ReactNode {
  if (result.kind === "refused") return result.message;
  if (result.kind === "expired") {
    return <>Your session ended. <Link to="/admin/sign-in" className="underline">Sign in again.</Link></>;
  }
  return "We could not delete. Please try again.";
}

/**
 * "Delete <heading>?" — and the shop's refusal inside the same question: an
 * order that is not cancelled, or is paid, stays and the question stays open
 * with the sentence under it, so the person reads why and nothing is lost.
 */
export default function OrderDelete({ open, id, title, onClose, onDeleted }: {
  open: boolean;
  id: string;
  title: string;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const [said, setSaid] = useState<ReactNode>(null);
  const [busy, setBusy] = useState(false);

  const close = () => {
    setSaid(null);
    onClose();
  };

  async function confirm() {
    setBusy(true);
    setSaid(null);
    const result = await deleteOrder(id);
    setBusy(false);
    if (result.ok) {
      close();
      onDeleted();
      return;
    }
    setSaid(refusal(result));
  }

  return (
    <AlertDialog open={open} onOpenChange={(next) => { if (!next && !busy) close(); }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="font-body">{`Delete ${title}?`}</AlertDialogTitle>
          <AlertDialogDescription>This cannot be undone.</AlertDialogDescription>
        </AlertDialogHeader>
        {said && <p role="alert" className="text-sm text-danger">{said}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={busy}
            onClick={(e) => { e.preventDefault(); void confirm(); }}>Delete</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
