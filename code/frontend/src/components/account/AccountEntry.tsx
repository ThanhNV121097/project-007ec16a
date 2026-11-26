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
 *
 * The bar's link cuts a long name with `truncate`, which clips whatever falls
 * outside its padding box, and the underline sits 8px below the line
 * (underline-offset-8). So the link carries 12px of padding under the line,
 * pulled back by the same margin: the underline stays inside the clipped box
 * and the link takes no more height than its neighbours.
 *
 * While /v1/auth/me is asked the bar holds the width of "Sign in" with an
 * invisible, aria-hidden copy, so the five links do not slide left when the
 * answer arrives; it shows no name and no "Sign in" to anyone.
 */
export default function AccountEntry({ variant, onNavigate }: { variant: "bar" | "menu"; onNavigate?: () => void }) {
  const { checking, me } = useMe();
  const { pathname } = useLocation();
  if (checking) {
    return variant === "bar"
      ? <span aria-hidden="true" data-testid="account-entry-pending" className="invisible block max-w-32 truncate">{W.entry.signIn}</span>
      : null;
  }
  const to = me ? homeFor(me) : "/account/sign-in";
  const current = pathname === to;
  const style = variant === "bar"
    ? `-mb-3 block max-w-32 truncate pb-3 text-ink underline-offset-8 hover:underline ${current ? "underline" : ""}`
    : "block truncate text-ink";
  return (
    <Link to={to} onClick={onNavigate} title={me?.name} aria-current={current ? "page" : undefined} data-testid="account-entry" className={style}>
      {me ? me.name : W.entry.signIn}
    </Link>
  );
}
