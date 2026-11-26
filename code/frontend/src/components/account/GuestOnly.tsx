import { type ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { homeFor, useMe } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";

/**
 * Sign-up and sign-in are for someone who is not signed in. While the shop is
 * asked who is signed in nothing is drawn, so a signed-in customer never sees
 * the form flash; once it answers, a signed-in person is taken home
 * (a customer to /account, the shop's team to /admin) and a visitor gets the
 * form. A successful sign-up or sign-in lands here too: asking /v1/auth/me
 * again is what moves the address.
 */
export default function GuestOnly({ children }: { children: ReactNode }) {
  const { checking, me } = useMe();
  if (checking) return null;
  return me ? <Navigate to={homeFor(me)} replace /> : <>{children}</>;
}
