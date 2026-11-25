import { Navigate, useNavigate } from "react-router-dom";
import { useContent } from "../../editable";
import { hello, W } from "../../lib/account-words";
import { customerSignOut, useMe } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import HeadPicture from "../HeadPicture";
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
  const pictureAlt = useContent("pages.track.head_alt");
  // The address changes first, so the page is gone before the shop answers and
  // /account never bounces a signing-out customer to the sign-in form.
  const leave = () => {
    navigate("/");
    void customerSignOut();
  };
  return (
    <>
      <section className="grid gap-x-8 gap-y-8 pb-10 pt-8 lg:grid-cols-12 lg:pt-10">
        <div className="flex min-w-0 flex-col items-start gap-5 lg:col-span-6 lg:justify-end">
          <h1 className="max-w-full wrap-anywhere text-[length:var(--size-h1)] leading-[1.02] text-ink">{W.account.title}</h1>
          <p className="max-w-full wrap-anywhere text-lg text-ink-soft">{hello(name)}</p>
          <button type="button" onClick={leave} className={OUTLINE}>{W.account.signOut}</button>
        </div>
        <HeadPicture src="/images/head-track.jpg" alt={pictureAlt} className="lg:col-span-5 lg:col-start-8" />
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
