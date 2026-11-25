import { type ReactNode } from "react";
import { Link, Navigate } from "react-router-dom";
import { useAuth } from "../../lib/session";

/**
 * THE DOOR OF EVERY /admin PAGE, and it opens only on the server's word.
 *
 * What it knows is the answer to "who is this?" (src/lib/session.ts) and
 * nothing the browser keeps for itself. Three outcomes, and a fourth moment
 * before them:
 *
 *   while the answer is awaited  nothing at all — no menu, no heading, no
 *                                redirect. A page drawn first and taken away
 *                                is a page a stranger has seen.
 *   nobody signed in             the sign-in page. A server that cannot be
 *                                reached is nobody, too.
 *   the shop's team              the page. That is an admin or a staff account.
 *   a customer                   a plain message and a way home. The pages are
 *                                never drawn, so no data call is sent for them.
 *
 * The server repeats every one of these checks on its own routes: this gate is
 * a courtesy to the person, never the defence.
 */
export default function AdminGate({ children }: { children: ReactNode }) {
  const { checking, session } = useAuth();
  if (checking) return null;
  if (!session) return <Navigate to="/admin/sign-in" replace />;
  if (session.role === "admin" || session.role === "staff") return <>{children}</>;
  return <TeamOnly />;
}

/** A customer's answer: text and a link, in the management palette, with none of the area. */
function TeamOnly() {
  return (
    <div data-dashboard="true" className="grid min-h-svh place-items-center bg-ground px-6 font-body text-ink">
      <main className="flex max-w-sm flex-col items-center gap-3 text-center">
        <p className="text-base font-medium text-ink">This area is for the shop's team</p>
        <Link to="/"
          className="inline-flex h-9 items-center rounded-sm px-2 text-sm font-medium text-accent underline underline-offset-4 outline-none focus-visible:ring-2 focus-visible:ring-accent">
          Back to the shop
        </Link>
      </main>
    </div>
  );
}
