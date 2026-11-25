import { Link, useLocation } from "react-router-dom";
import { W } from "../../lib/account-words";
import { homeFor, useMe } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";

/**
 * THE HEADER'S ACCOUNT ENTRY: "Sign in" for a visitor, the customer's name
 * for a customer, the team member's name for staff and admin (who go to
 * /admin). Its state is what GET /v1/auth/me says and nothing else; while
 * that is being asked nothing is drawn, and a 401 or an unreachable shop reads
 * "Sign in" with no error. The name is cut with an ellipsis so it never pushes
 * the search and bag buttons off the bar.
 *
 * `bar` is the link of the header from 1024px (text-sm, underline on hover
 * and on its own address); `menu` is the menu's link below it.
 */
export default function AccountEntry({ variant, onNavigate }: { variant: "bar" | "menu"; onNavigate?: () => void }) {
  const { checking, me } = useMe();
  const { pathname } = useLocation();
  if (checking) return null;
  const to = me ? homeFor(me) : "/account/sign-in";
  const current = pathname === to;
  const style = variant === "bar"
    ? `block max-w-32 truncate text-ink underline-offset-8 hover:underline ${current ? "underline" : ""}`
    : "block truncate text-ink";
  return (
    <Link to={to} onClick={onNavigate} title={me?.name} aria-current={current ? "page" : undefined} data-testid="account-entry" className={style}>
      {me ? me.name : W.entry.signIn}
    </Link>
  );
}
