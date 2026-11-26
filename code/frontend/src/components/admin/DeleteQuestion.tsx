import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "../ui/alert-dialog";
import { FailureLine } from "./SaveNote";
import type { Outcome } from "./store";
import { useAction } from "./useSave";

/**
 * ASKING BEFORE SOMETHING IS GONE, and staying until the shop has answered.
 *
 * The question names the record. Confirming sends the delete and keeps the
 * question open: when the shop refuses — "Orders refer to this shoe. Set it to
 * draft instead." — the sentence is shown inside it, beside the buttons that
 * were pressed, and nothing is closed behind the person's back. Mounted only
 * while it is asked, so every question starts clean.
 */
export default function DeleteQuestion({ title, onClose, onDelete, onDeleted }: {
  title: string;
  onClose: () => void;
  onDelete: () => Promise<Outcome<unknown>>;
  onDeleted: () => void;
}) {
  const act = useAction();

  async function confirm(e: React.MouseEvent) {
    // The shop has not answered: the question must not close itself.
    e.preventDefault();
    const out = await act.run(onDelete);
    if (out?.ok) onDeleted();
  }

  return (
    <AlertDialog open onOpenChange={(next) => { if (!next && !act.busy) onClose(); }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="font-body">{title}</AlertDialogTitle>
          <AlertDialogDescription>This cannot be undone.</AlertDialogDescription>
        </AlertDialogHeader>
        {act.failure && (
          <p role="alert" className="text-sm text-danger"><FailureLine failure={act.failure} what="delete" /></p>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={act.busy}>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={act.busy} onClick={(e) => void confirm(e)}>
            {act.busy ? "Deleting…" : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
