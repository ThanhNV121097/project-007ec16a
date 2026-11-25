import { useRef, useState, type ChangeEvent, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { checkSignUp, type Problems } from "../../lib/account-rules";
import { W } from "../../lib/account-words";
import { customerSignUp, type Me, type Outcome } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import AuthLayout from "./AuthLayout";
import Field, { ErrorLine } from "./Field";
import { useFocusFirstInvalid } from "./hooks";
import { FILLED, TEXT_LINK } from "./styles";

const EMPTY = { name: "", email: "", phone: "", password: "" };
type Values = typeof EMPTY;

/** What the page says about a refused sign-up: a sentence under a field, or one line above the button. */
function refusal(out: Exclude<Outcome<Me>, { kind: "ok" }>): { problems: Problems; notice: string } {
  switch (out.kind) {
    case "conflict":
      return { problems: { email: W.signUp.taken }, notice: "" };
    case "invalid":
      return Object.keys(out.errors).length > 0 ? { problems: out.errors, notice: "" } : { problems: {}, notice: W.unreachable };
    case "limited":
      return { problems: {}, notice: W.signUp.limited };
    default:
      return { problems: {}, notice: W.unreachable };
  }
}

/**
 * SIGN UP: name, email, phone and password; checked here first (nothing is
 * sent while a field is wrong), then by the shop. After a refusal the typed
 * values stay and the password is emptied. On success the shop has signed the
 * customer in and GuestOnly takes the address to /account.
 */
export default function SignUpView() {
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
    const found = checkSignUp(values);
    setProblems(found);
    setNotice("");
    if (Object.keys(found).length > 0) return;
    sending.current = true;
    setBusy(true);
    const out = await customerSignUp({ ...values, name: values.name.trim(), email: values.email.trim(), phone: values.phone.trim() });
    sending.current = false;
    setBusy(false);
    if (out.kind === "ok") return;
    const said = refusal(out);
    setValues((v) => ({ ...v, password: "" }));
    setProblems(said.problems);
    setNotice(said.notice);
  }

  return (
    <AuthLayout title={W.signUp.title} lead={W.signUp.lead}>
      <form ref={form} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <Field id="sign-up-name" label={W.labels.name} autoComplete="name" {...bind("name")} />
        <Field id="sign-up-email" label={W.labels.email} type="email" autoComplete="email" {...bind("email")} />
        <Field id="sign-up-phone" label={W.labels.phone} type="tel" inputMode="tel" autoComplete="tel" {...bind("phone")} />
        <Field id="sign-up-password" label={W.labels.password} type="password" autoComplete="new-password" {...bind("password")} />
        {notice && <ErrorLine>{notice}</ErrorLine>}
        <button type="submit" disabled={busy} className={FILLED}>{busy ? W.signUp.busy : W.signUp.submit}</button>
      </form>
      <p className="mt-6 text-sm text-ink-soft">
        {W.signUp.have} <Link to="/account/sign-in" className={TEXT_LINK}>{W.signUp.haveLink}</Link>
      </p>
    </AuthLayout>
  );
}
