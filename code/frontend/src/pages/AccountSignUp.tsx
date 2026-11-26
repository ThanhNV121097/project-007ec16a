import GuestOnly from "../components/account/GuestOnly";
import SignUpView from "../components/account/SignUpView";

/** /account/sign-up: make a customer account. Its words live in the components (src/lib/account-words.ts). */
export default function AccountSignUp() {
  return <GuestOnly><SignUpView /></GuestOnly>;
}
