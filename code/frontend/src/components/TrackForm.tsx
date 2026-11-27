import { useEffect, useRef, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { T, useContent } from "../editable";
import { lookupOrder, type LookupFailure } from "../lib/track-an-order-by-number-and-phone";
import Icon from "./Icon";

const RING = "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink";
const FIELD = `w-full appearance-none rounded-sm border border-field bg-ground px-3 py-3 text-base text-ink placeholder:text-ink-soft focus:border-ink sm:text-sm ${RING}`;
const LABEL = "mb-1.5 block text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft";
const PILL = `mt-2 inline-flex items-center justify-center gap-2 self-start rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85 disabled:opacity-60 ${RING}`;

/** The content key of the sentence each way a lookup can fail is told in. */
const SENTENCE: Record<LookupFailure, string> = {
  not_found: "pages.track.not_found",
  too_many: "pages.track.too_many",
  offline: "pages.track.offline",
  failed: "pages.track.failed",
};

/**
 * THE TRACK FORM: the order number and the phone it was placed with, sent to
 * the shop as typed. The page checks nothing of its own, so nothing on it says
 * which half was wrong; a refusal is one line under the fields, one at a time,
 * and what was typed stays. A right pair opens the order's own page.
 */
export default function TrackForm() {
  const navigate = useNavigate();
  const placeholder = useContent("pages.track.placeholder");
  const searchingWords = useContent("pages.track.searching");
  const [number, setNumber] = useState("");
  const [phone, setPhone] = useState("");
  const [problem, setProblem] = useState<LookupFailure | null>(null);
  const [searching, setSearching] = useState(false);
  const sentence = useContent(SENTENCE[problem ?? "not_found"]);
  const asking = useRef(false);
  const here = useRef(true);

  useEffect(() => {
    here.current = true;
    return () => {
      here.current = false;
    };
  }, []);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (asking.current) return;
    asking.current = true;
    setProblem(null);
    setSearching(true);
    const result = await lookupOrder(number, phone);
    asking.current = false;
    if (!here.current) return;
    if (result.ok) {
      navigate(`/orders/${result.order.id}`);
      return;
    }
    setSearching(false);
    setProblem(result.kind);
  };

  return (
    <form onSubmit={(e) => void submit(e)} noValidate className="flex max-w-md flex-col gap-4">
      <div>
        <label htmlFor="order-number" className={LABEL}><T k="pages.track.number" /></label>
        <input id="order-number" value={number} onChange={(e) => setNumber(e.target.value)} placeholder={placeholder} className={FIELD} />
      </div>
      <div>
        <label htmlFor="order-phone" className={LABEL}><T k="pages.track.phone" /></label>
        <input id="order-phone" type="tel" inputMode="tel" value={phone} onChange={(e) => setPhone(e.target.value)} className={FIELD} />
      </div>
      {problem && (
        <p role="alert" data-testid="track-error" className="flex items-center gap-2 text-xs text-danger">
          <Icon name="alert" size={14} className="shrink-0" />{sentence}
        </p>
      )}
      <button type="submit" data-testid="track-submit" disabled={searching} className={PILL}>
        {searching ? searchingWords : <><T k="pages.track.submit" /><Icon name="arrow" size={16} /></>}
      </button>
    </form>
  );
}
