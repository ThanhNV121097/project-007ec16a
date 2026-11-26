/**
 * THE WAY INTO THE MANAGEMENT AREA.
 *
 * Every CMS needs one and no two of them differ: an address, a password, one
 * button. It is also the screen a design turn is most likely to get subtly
 * wrong at the end of a long build — a form that is not a <form> (so Enter
 * does nothing), a password field that is a text field, an area whose pages
 * are gated while the door into them is not.
 *
 *   <SignInForm to="/admin" />          the form alone, wherever you put it
 *   <SignInPage title=… to="/admin" />  the whole screen, for when you do not
 *
 * The form asks the shop's server (src/lib/session.ts) and the server decides
 * who the person is and what they may open: the form carries no role. Its words
 * for what can go wrong live here, beside the states they belong to.
 */
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Icon from "../Icon";
import { Button, Field, Input } from "./index";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { signIn, useSession } from "../../lib/session";

/** What the shop said, as the one line under the fields. It never says which part was wrong. */
const PROBLEMS = {
  wrong: "Wrong email or password.",
  limited: "Too many tries. Please wait 15 minutes.",
  unreachable: "We cannot reach the server. Please try again.",
} as const;

type Problem = keyof typeof PROBLEMS | "";

interface Missing {
  email?: string;
  password?: string;
}

/** What is left blank, stopped here: nothing is sent for it. */
function missing(email: string, password: string): Missing {
  if (!email.trim()) return { email: "An email address is needed" };
  if (!password) return { password: "A password is needed" };
  return {};
}

/** The form alone: no card, no screen. Carries the management palette with
 *  it, so it is right wherever a design places it. */
export function SignInForm({ to = "/", note }: {
  to?: string;
  note?: React.ReactNode;
}) {
  const navigate = useNavigate();
  const session = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [blank, setBlank] = useState<Missing>({});
  const [problem, setProblem] = useState<Problem>("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy) return;
    const left = missing(email, password);
    setBlank(left);
    setProblem("");
    const first = left.email ? "email" : left.password ? "password" : "";
    if (first) {
      (e.currentTarget.elements.namedItem(first) as HTMLInputElement | null)?.focus();
      return;
    }
    setBusy(true);
    const result = await signIn(email, password);
    setBusy(false);
    if (result === "ok") {
      navigate(to);
      return;
    }
    setProblem(result);
    // The password is never kept after a refusal; unreachable keeps it, so
    // one press tries again.
    if (result !== "unreachable") setPassword("");
  }

  return (
    <div data-dashboard="true" className="bg-transparent">
      {/* A <form>, so Enter in either field signs in — the one thing
          everybody tries and the commonest omission of a hand-written one. */}
      <form onSubmit={submit} noValidate className="grid gap-5">
        <Field label="Email" error={blank.email}>
          <Input name="email" type="email" autoComplete="username" placeholder="you@example.com" value={email}
            aria-invalid={blank.email ? true : undefined}
            onChange={(e) => { setEmail(e.target.value); setBlank({}); }} />
        </Field>
        <Field label="Password" error={blank.password}>
          <Input name="password" type="password" autoComplete="current-password" value={password}
            aria-invalid={blank.password ? true : undefined}
            onChange={(e) => { setPassword(e.target.value); setBlank({}); }} />
        </Field>
        {problem && (
          <p role="alert" className="flex items-start gap-2 text-sm text-danger">
            <Icon name="alert" size={16} className="mt-0.5 shrink-0" />
            {PROBLEMS[problem]}
          </p>
        )}
        <Button kind="primary" type="submit" disabled={busy} className="h-9 w-full">
          {busy ? "Signing in…" : "Sign in"}
        </Button>
      </form>
      {note && <p className="mt-4 text-center text-sm text-ink-soft">{note}</p>}
      {session && (
        <p className="mt-4 text-center text-sm text-ink-soft">
          Signed in as {session.email}.{" "}
          <button type="button" onClick={() => navigate(to)} className="font-medium text-ink underline underline-offset-4">
            Continue
          </button>
        </p>
      )}
    </div>
  );
}

export default function SignInPage({ title, description, to = "/", note, brand }: {
  title: string;
  /** One line under the title: what this area is. */
  description?: string;
  /** Where signing in lands. The area's overview. */
  to?: string;
  /** A line under the button: who this area is for, who to ask for access. */
  note?: React.ReactNode;
  brand?: React.ReactNode;
}) {
  return (
    <div data-dashboard="true" className="flex min-h-svh flex-col items-center justify-center gap-6 bg-ground p-6 font-body md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-6">
        {brand && <div className="flex items-center justify-center gap-2 font-medium text-ink">{brand}</div>}
        <Card className="shadow-xs">
          <CardHeader className="text-center">
            <CardTitle className="font-body text-xl">{title}</CardTitle>
            {description && <CardDescription>{description}</CardDescription>}
          </CardHeader>
          <CardContent>
            <SignInForm to={to} note={note} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
