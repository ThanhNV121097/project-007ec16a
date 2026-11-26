import { useRef, useState, type ChangeEvent, type FormEvent } from "react";
import { checkDetails, type Problems } from "../../lib/account-rules";
import { W } from "../../lib/account-words";
import { refreshMe, saveAccount, useSavedDetailsFill, type Details } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import AccountBlock from "./AccountBlock";
import Field, { ErrorLine } from "./Field";
import { useFocusFirstInvalid } from "./hooks";
import { FILLED } from "./styles";

type Phase = "idle" | "saving" | "saved" | "failed";

/**
 * SAVED DELIVERY DETAILS: the checkout's four fields, kept for next time. The
 * form is usable at once, with the name the shop already knows; the rest
 * fills in when GET /v1/account arrives, but only fields still empty, so what
 * was typed meanwhile is never overwritten. A refusal from the shop shows its
 * sentence under the field; a failure keeps everything typed.
 */
export default function SavedDetails({ name }: { name: string }) {
  const form = useRef<HTMLFormElement>(null);
  const sending = useRef(false);
  const [values, setValues] = useState<Details>({ name, phone: "", address: "", city: "" });
  const [problems, setProblems] = useState<Problems>({});
  const [phase, setPhase] = useState<Phase>("idle");
  useSavedDetailsFill(setValues);
  useFocusFirstInvalid(form, problems);

  const bind = (key: keyof Details) => ({
    value: values[key],
    error: problems[key],
    onChange: (e: ChangeEvent<HTMLInputElement>) => {
      setValues((v) => ({ ...v, [key]: e.target.value }));
      setPhase((p) => (p === "saving" ? p : "idle"));
    },
  });

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (sending.current) return;
    const found = checkDetails(values);
    setProblems(found);
    if (Object.keys(found).length > 0) return;
    sending.current = true;
    setPhase("saving");
    const out = await saveAccount({ name: values.name.trim(), phone: values.phone.trim(), address: values.address.trim(), city: values.city.trim() });
    sending.current = false;
    if (out.kind === "ok") {
      setValues({ name: out.value.name, phone: out.value.phone, address: out.value.address, city: out.value.city });
      setPhase("saved");
      return;
    }
    if (out.kind === "unauthenticated" || out.kind === "forbidden") void refreshMe();
    if (out.kind === "invalid" && Object.keys(out.errors).length > 0) setProblems(out.errors);
    setPhase(out.kind === "invalid" && Object.keys(out.errors).length > 0 ? "idle" : "failed");
  }

  return (
    <AccountBlock id="saved-details" title={W.details.title}>
      <form ref={form} noValidate onSubmit={(e) => void submit(e)} aria-labelledby="saved-details-title" className="flex max-w-md flex-col gap-4">
        <Field id="details-name" label={W.details.name} autoComplete="name" {...bind("name")} />
        <Field id="details-phone" label={W.details.phone} type="tel" inputMode="tel" autoComplete="tel" {...bind("phone")} />
        <Field id="details-address" label={W.details.address} autoComplete="street-address" {...bind("address")} />
        <Field id="details-city" label={W.details.city} autoComplete="address-level1" {...bind("city")} />
        {phase === "failed" && <ErrorLine>{W.details.failed}</ErrorLine>}
        <div className="flex flex-wrap items-center gap-x-4">
          <button type="submit" disabled={phase === "saving"} className={FILLED}>{phase === "saving" ? W.details.saving : W.details.save}</button>
          <p role="status" className="text-sm text-ink">{phase === "saved" ? W.details.saved : ""}</p>
        </div>
      </form>
    </AccountBlock>
  );
}
