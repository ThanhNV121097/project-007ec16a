import { useRef, useState, type ChangeEvent, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { checkSignIn, type Problems } from "../../lib/account-rules";
import { W } from "../../lib/account-words";
import { customerSignIn, type Me, type Outcome } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import AuthLayout from "./AuthLayout";
import Field, { ErrorLine } from "./Field";
import { useFocusFirstInvalid } from "./hooks";
import { FILLED, TEXT_LINK } from "./styles";

const EMPTY = { email: "", password: "" };
type Values = typeof EMPTY;

/** One sentence for a wrong email and a wrong password alike, so the page never says which one was wrong. */
function noticeFor(out: Exclude<Outcome<Me>, { kind: "ok" }>): string {
  switch (out.kind) {
    case "unauthenticated":
    case "notfound":
      return W.signIn.wrong;
    case "limited":
      return W.signIn.limited;
    default:
      return W.unreachable;
  }
}

/**
 * SIGN IN, for a customer, through the shared sign-in. A blank field is named
 * here and nothing is sent; the pair is judged only by the shop. After any
 * refusal the email stays and the password is emptied. On success GuestOnly
 * takes a customer to /account and the shop's team to /admin.
 */
export default function SignInView() {
  const form = useRef<HTMLFormElement>(null);
  const sending = useRef(false);
  const [values, setValues] = useState<Values>(EMPTY);
  const [problems, setProblems] = useState<Problems>({});
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  useFocusFirstInvalid(form, problems);

  const bind = (key: keyof Values) => ({
    value: values[key],
    error: problems[key],
    onChange: (e: ChangeEvent<HTMLInputElement>) => setValues((v) => ({ ...v, [key]: e.target.value })),
  });

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (sending.current) return;
    const found = checkSignIn(values);
    setProblems(found);
    setNotice("");
    if (Object.keys(found).length > 0) return;
    sending.current = true;
    setBusy(true);
    const out = await customerSignIn(values.email.trim(), values.password);
    sending.current = false;
    setBusy(false);
    if (out.kind === "ok") return;
    setValues((v) => ({ ...v, password: "" }));
    setNotice(noticeFor(out));
  }

  return (
    <AuthLayout title={W.signIn.title} art="track">
      <form ref={form} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <Field id="sign-in-email" label={W.labels.email} type="email" autoComplete="username" {...bind("email")} />
        <Field id="sign-in-password" label={W.labels.password} type="password" autoComplete="current-password" {...bind("password")} />
        {notice && <ErrorLine>{notice}</ErrorLine>}
        <button type="submit" disabled={busy} className={FILLED}>{busy ? W.signIn.busy : W.signIn.submit}</button>
      </form>
      <p className="mt-6 text-sm text-ink-soft">
        {W.signIn.fresh} <Link to="/account/sign-up" className={TEXT_LINK}>{W.signIn.freshLink}</Link>
      </p>
      <p className="mt-2 text-sm text-ink-soft">{W.signIn.guest}</p>
    </AuthLayout>
  );
}
