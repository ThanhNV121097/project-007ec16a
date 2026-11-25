import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { formatDate } from "../lib/format";
import { usePlacedOrders } from "../lib/mock/orders";
import { digitsOf, ORDER_STEPS, type Order } from "../lib/shop";
import Disclosure from "../components/Disclosure";
import Icon from "../components/Icon";
import Rule from "../components/Rule";

const FIELD = "w-full appearance-none rounded-sm border border-field bg-ground px-3 py-3 text-base text-ink outline-none placeholder:text-ink-soft focus:border-ink sm:text-sm";
const LABEL = "mb-1.5 block text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft";

/**
 * TRACK AN ORDER: the order number and the phone it was placed with, and
 * the answer is the order's own page. Beside the form, what each of the
 * five steps means — the question a buyer asks while waiting.
 */
export default function Track() {
  usePageMeta("pages.track.title", "pages.track.description");
  const navigate = useNavigate();
  const orders = useCollection<Order>("orders", { pageSize: 500 }).items;
  const placed = usePlacedOrders();
  const notFound = useContent("pages.track.not_found");
  const placeholder = useContent("pages.track.placeholder");
  const steps = useContent<Record<string, string>>("pages.track.steps");
  const labels = useContent<Record<string, string>>("orders.steps");
  const [number, setNumber] = useState("");
  const [phone, setPhone] = useState("");
  const [error, setError] = useState("");

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const id = `doi-${number.toLowerCase().replace(/[^0-9]/g, "")}`;
    const typed = digitsOf(phone);
    const found = orders.find((o) => o.id === id) ?? placed.find((o) => o.id === id);
    const matches = found && typed.length >= 4 && digitsOf(found.phone).endsWith(typed.slice(-4));
    if (!matches) {
      setError(notFound);
      return;
    }
    navigate(`/orders/${id}`);
  };

  return (
    <>
      <section className="grid gap-12 pb-12 pt-8 lg:grid-cols-12 lg:gap-8 lg:pt-10">
        <div className="lg:col-span-6">
          <T k="pages.track.heading" as="h1" className="max-w-[10ch] text-[length:var(--size-h1)] leading-[1.02] text-ink" />
          <T k="pages.track.lead" as="p" className="mt-5 max-w-[42ch] text-sm leading-relaxed text-ink-soft" />
          <form onSubmit={submit} noValidate className="mt-8 flex max-w-md flex-col gap-4">
            <div><label htmlFor="order-number" className={LABEL}><T k="pages.track.number" /></label>
              <input id="order-number" value={number} onChange={(e) => setNumber(e.target.value)} placeholder={placeholder} className={FIELD} /></div>
            <div><label htmlFor="order-phone" className={LABEL}><T k="pages.track.phone" /></label>
              <input id="order-phone" type="tel" inputMode="tel" value={phone} onChange={(e) => setPhone(e.target.value)} className={FIELD} /></div>
            {error && <p role="alert" className="flex items-center gap-2 text-xs text-danger"><Icon name="alert" size={14} className="shrink-0" />{error}</p>}
            <button type="submit" data-testid="track-submit"
              className="mt-2 inline-flex items-center justify-center gap-2 self-start rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85">
              <T k="pages.track.submit" /><Icon name="arrow" size={16} />
            </button>
          </form>
        </div>
        <div className="lg:col-span-5 lg:col-start-8">
          <T k="pages.track.steps_heading" as="h2" className="mb-5 text-lg text-ink" />
          <Disclosure rows={ORDER_STEPS.map((s) => ({ label: labels?.[s] ?? s, content: <p className="max-w-[48ch] leading-relaxed">{steps?.[s]}</p> }))} />
          {placed.length > 0 && (
            <div className="mt-10">
              <T k="pages.track.yours" as="h2" className="mb-3 text-lg text-ink" />
              <ul className="border-t border-line">
                {placed.slice(0, 5).map((o) => (
                  <li key={o.id} className="border-b border-line">
                    <Link to={`/orders/${o.id}`} className="flex items-center justify-between gap-4 py-3 text-sm text-ink hover:underline">
                      <span className="tabular-nums">{o.number}</span>
                      <span className="text-xs tabular-nums text-ink-soft">{formatDate(o.placed_at, "medium")}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </section>
      <Rule />
    </>
  );
}
