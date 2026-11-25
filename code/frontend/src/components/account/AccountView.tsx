import { Navigate, useNavigate } from "react-router-dom";
import { hello, W } from "../../lib/account-words";
import { customerSignOut, useMe } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import AddOrder from "./AddOrder";
import { usePageTitle } from "./hooks";
import MyOrders from "./MyOrders";
import SavedDetails from "./SavedDetails";
import { OUTLINE } from "./styles";
import { useMyOrders } from "./useMyOrders";

/** The page of a signed-in customer: who they are, their orders, their details, an order to add. */
function SignedIn({ name }: { name: string }) {
  usePageTitle(W.account.title);
  const navigate = useNavigate();
  const orders = useMyOrders();
  // The address changes first, so the page is gone before the shop answers and
  // /account never bounces a signing-out customer to the sign-in form.
  const leave = () => {
    navigate("/");
    void customerSignOut();
  };
  return (
    <>
      <section className="flex flex-col items-start gap-5 pb-10 pt-8 lg:pt-10">
        <h1 className="max-w-full wrap-anywhere text-[length:var(--size-h1)] leading-[1.02] text-ink">{W.account.title}</h1>
        <p className="max-w-full wrap-anywhere text-lg text-ink-soft">{hello(name)}</p>
        <button type="button" onClick={leave} className={OUTLINE}>{W.account.signOut}</button>
      </section>
      <MyOrders orders={orders} />
      <SavedDetails name={name} />
      <AddOrder onAdded={orders.reload} />
    </>
  );
}

/**
 * MY ACCOUNT. While /v1/auth/me is asked, nothing is drawn and nothing moves;
 * then a visitor goes to the sign-in form, the shop's team to /admin, and a
 * customer sees the page.
 */
export default function AccountView() {
  const { checking, me } = useMe();
  if (checking) return null;
  if (!me) return <Navigate to="/account/sign-in" replace />;
  if (me.role !== "customer") return <Navigate to="/admin" replace />;
  return <SignedIn name={me.name} />;
}
