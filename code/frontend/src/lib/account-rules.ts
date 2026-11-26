import { W } from "./account-words";

/** Field name → the sentence to show under it. An empty object is a form that may be sent. */
export type Problems = Record<string, string>;

const digitsIn = (s: string): number => s.replace(/\D/g, "").length;

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Each rule answers "" when the value is fine, else the sentence. The shop checks the same limits again. */
function nameProblem(value: string): string {
  const name = value.trim();
  if (!name) return W.rules.name;
  return name.length > 100 ? W.rules.max100 : "";
}

function emailProblem(value: string): string {
  const email = value.trim();
  return EMAIL.test(email) && email.length <= 200 ? "" : W.rules.email;
}

/** At least 9 digits (spaces, +, - and brackets are not counted), at most 20 characters: the checkout's rule. */
function phoneProblem(value: string): string {
  const phone = value.trim();
  return digitsIn(phone) >= 9 && phone.length <= 20 ? "" : W.rules.phone;
}

/** 8 to 100 characters, never trimmed. */
function passwordProblem(value: string): string {
  if (value.length < 8) return W.rules.passwordShort;
  return value.length > 100 ? W.rules.max100 : "";
}

function collect(found: [string, string][]): Problems {
  const out: Problems = {};
  for (const [field, sentence] of found) if (sentence) out[field] = sentence;
  return out;
}

export function checkSignUp(v: { name: string; email: string; phone: string; password: string }): Problems {
  return collect([
    ["name", nameProblem(v.name)],
    ["email", emailProblem(v.email)],
    ["phone", phoneProblem(v.phone)],
    ["password", passwordProblem(v.password)],
  ]);
}

/** Blank fields only: whether the pair is right is the shop's answer, never the page's. */
export function checkSignIn(v: { email: string; password: string }): Problems {
  return collect([
    ["email", v.email.trim() ? "" : W.signIn.needEmail],
    ["password", v.password ? "" : W.signIn.needPassword],
  ]);
}

export function checkDetails(v: { name: string; phone: string; address: string; city: string }): Problems {
  return collect([
    ["name", nameProblem(v.name)],
    ["phone", phoneProblem(v.phone)],
    ["address", v.address.trim().length > 200 ? W.rules.max200 : ""],
    ["city", v.city.trim().length > 100 ? W.rules.max100 : ""],
  ]);
}
