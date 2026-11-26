import GuestOnly from "../components/account/GuestOnly";
import SignInView from "../components/account/SignInView";

/** /account/sign-in: the customer's way in, through the shared sign-in. */
export default function AccountSignIn() {
  return <GuestOnly><SignInView /></GuestOnly>;
}
