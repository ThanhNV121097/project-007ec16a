import { Link } from "react-router-dom";
import { Notice } from "./index";
import type { Failure } from "./store";

/**
 * WHAT THE SHOP SAID WHEN A SAVE DID NOT HAPPEN, in the person's words.
 *
 * One sentence per kind of refusal, the same wherever it appears — a dialog, a
 * record page, a delete question, a picture, one size's count — so the person
 * learns them once: the shop's own sentence where it gave one, "your session
 * ended" with the way back in, and "we could not… please try again" for an
 * answer that never came.
 */
const UNREACHABLE = {
  save: "We could not save. Please try again.",
  delete: "We could not delete. Please try again.",
  upload: "We could not upload that picture. Please try again.",
} as const;

export function FailureLine({ failure, what = "save" }: { failure: Failure; what?: keyof typeof UNREACHABLE }) {
  switch (failure.kind) {
    case "session":
      return (
        <>
          Your session ended.{" "}
          <Link to="/admin/sign-in"
            className="rounded-sm underline underline-offset-4 outline-none focus-visible:ring-2 focus-visible:ring-accent">
            Sign in again.
          </Link>
        </>
      );
    case "message":
    case "stale":
      return <>{failure.message}</>;
    case "fields":
      return <>{Object.values(failure.errors).join(" ")}</>;
    default:
      return <>{UNREACHABLE[what]}</>;
  }
}

/**
 * The note above a form. The shop's field errors sit beside their fields, so
 * of those it shows only the ones no field on this form can show; every other
 * refusal is the note.
 */
export default function SaveNote({ failure, placed }: { failure: Failure | null; placed: ReadonlySet<string> }) {
  if (!failure) return null;
  if (failure.kind === "fields") {
    const loose = Object.entries(failure.errors).filter(([key]) => !placed.has(key) && !key.startsWith("pairs."));
    if (loose.length === 0) return null;
    return <Notice tone="alert">{loose.map(([, sentence]) => sentence).join(" ")}</Notice>;
  }
  return <Notice tone="alert"><FailureLine failure={failure} /></Notice>;
}
