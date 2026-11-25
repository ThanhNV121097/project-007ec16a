import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { clearCart, lineKey, useCart, type CartLine } from "../lib/cart";
import { useDemo } from "../lib/demo";
import { formatPrice } from "../lib/format";
import { placeOrder } from "../lib/mock/orders";
import { digitsOf, fill, type DeliveryOption, type PaymentMethod, type Product } from "../lib/shop";
import HeadPicture from "../components/HeadPicture";
import Icon, { type IconName } from "../components/Icon";
import EmptyBag from "../components/EmptyBag";
import ProductImage from "../components/ProductImage";
import Rule from "../components/Rule";

const FIELD = "w-full appearance-none rounded-sm border border-field bg-ground px-3 py-3 text-base text-ink outline-none placeholder:text-ink-soft focus:border-ink sm:text-sm";
const LABEL = "mb-1.5 block text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft";

const STEPS = ["pages.checkout.step_contact", "pages.checkout.step_delivery", "pages.checkout.step_payment"] as const;

function Step({ n, k }: { n: number; k: string }) {
  return (
    <legend className="mb-5 flex items-baseline gap-3 text-lg text-ink">
      <span className="text-sm tabular-nums text-ink-soft">{String(n).padStart(2, "0")}</span><T k={k} />
    </legend>
  );
}

/**
 * A row that is the only one there is: drawn selected, as the design draws a
 * chosen option, but with no control, because there is nothing to choose.
 */
function OnlyRow({ testid, icon, title, note, aside }: { testid: string; icon: IconName; title: string; note?: string; aside?: string }) {
  return (
    <div data-testid={testid} className="flex gap-4 border-b border-line py-4">
      <span aria-hidden="true" className="mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border border-ink"><span className="size-2 rounded-full bg-ink" /></span>
      <Icon name={icon} size={18} className="mt-0.5 shrink-0 text-ink" />
      <span className="min-w-0 flex-1">
        <span className="block text-sm text-ink">{title}</span>
        <span className="mt-1 block text-sm leading-relaxed text-ink-soft">{note}</span>
      </span>
      {aside && <span className="shrink-0 text-sm tabular-nums text-ink">{aside}</span>}
    </div>
  );
}

/**
 * CHECKOUT, ON ONE PAGE: who and where, delivery, how to pay — three
 * numbered groups because they are a sequence — with the pairs, their
 * pictures and the total kept in view beside them, and what happens after
 * "Place order" said before it is pressed. Payment is a mock: an online
 * method is marked paid at once; the real gateway arrives with the backend.
 */
export default function Checkout() {
  usePageMeta("pages.checkout.title", "pages.checkout.description");
  const navigate = useNavigate();
  const { lines: real } = useCart();
  const demo = useDemo();
  const products = useCollection<Product>("products", { where: { status: "live" }, pageSize: 500 }).items;
  const standard = useRecord<DeliveryOption>("delivery", "standard").item;
  const method = useRecord<PaymentMethod>("payments", "vnpay").item;
  const words = {
    free: useContent("pages.cart.free"),
    nextOnline: useContent("pages.checkout.next_online"),
    deliveryTitle: useContent("pages.checkout.delivery_title"),
    paying: useContent("pages.checkout.paying"),
    missing: useContent("pages.checkout.missing"),
    phoneBad: useContent("pages.checkout.phone_bad"),
    size: useContent("pages.checkout.line"),
    headAlt: useContent("pages.checkout.head_alt"),
  };
  const [form, setForm] = useState({ name: "", phone: "", address: "", city: "", note: "" });
  const [errors, setErrors] = useState<string[]>([]);
  const [paying, setPaying] = useState(false);

  const example: CartLine[] = products.slice(1, 3).map((p) => ({
    id: p.id, name: p.name, price: p.price, image: p.image, variant: `EU ${(p.sizes ?? [])[3] ?? ""}`, qty: 1,
  }));
  const showingExample = demo && real.length === 0;
  const lines = showingExample ? example : real;
  const subtotal = lines.reduce((s, l) => s + l.price * l.qty, 0);
  const set = (k: keyof typeof form) => (v: string) => setForm((f) => ({ ...f, [k]: v }));

  if (lines.length === 0) return <EmptyBag heading="pages.checkout.empty" />;

  const place = (e: FormEvent) => {
    e.preventDefault();
    const wrong: string[] = [];
    if (!form.name.trim() || !form.address.trim() || !form.city.trim()) wrong.push(words.missing);
    if (digitsOf(form.phone).length < 9) wrong.push(words.phoneBad);
    setErrors(wrong);
    if (wrong.length > 0) return;
    setPaying(true);
    window.setTimeout(() => {
      const order = placeOrder({
        lines, payment: "vnpay", online: true, customer: form.name.trim(), phone: form.phone.trim(),
        address: form.address.trim(), city: form.city.trim(), note: form.note.trim(),
      });
      if (!showingExample) clearCart();
      navigate(`/orders/${order.id}?placed=1`);
    }, 900);
  };

  return (
    <>
      <section className="grid gap-6 pb-8 pt-8 lg:grid-cols-12 lg:gap-8 lg:pt-10">
        <div className="flex flex-col justify-end gap-4 lg:col-span-6">
        <T k="pages.checkout.heading" as="h1" className="text-[length:var(--size-h1)] leading-[1.02] text-ink" />
        <ol className="hidden items-center gap-6 text-sm text-ink-soft lg:flex">
          {STEPS.map((k, i) => (
            <li key={k} className="flex items-baseline gap-2"><span className="tabular-nums">{String(i + 1).padStart(2, "0")}</span><T k={k} className="text-ink" /></li>
          ))}
        </ol>
        <p className="flex items-center gap-2 text-sm text-ink-soft"><Icon name="secure" size={16} className="shrink-0 text-ink" /><T k="pages.checkout.secure" /></p>
        </div>
        <HeadPicture src="/images/head-checkout.jpg" alt={words.headAlt} className="lg:col-span-6 lg:col-start-7" />
      </section>
      <Rule />
      <form onSubmit={place} noValidate className="grid gap-12 py-10 lg:grid-cols-12 lg:gap-8">
        <div className="flex flex-col gap-10 lg:col-span-7">
          <fieldset>
            <Step n={1} k="pages.checkout.step_contact" />
            <div className="grid gap-4 sm:grid-cols-2">
              <div><label htmlFor="checkout-name" className={LABEL}><T k="pages.checkout.name" /></label>
                <input id="checkout-name" autoComplete="name" value={form.name} onChange={(e) => set("name")(e.target.value)} className={FIELD} /></div>
              <div><label htmlFor="checkout-phone" className={LABEL}><T k="pages.checkout.phone" /></label>
                <input id="checkout-phone" type="tel" autoComplete="tel" inputMode="tel" value={form.phone} onChange={(e) => set("phone")(e.target.value)} className={FIELD} /></div>
              <div className="sm:col-span-2"><label htmlFor="checkout-address" className={LABEL}><T k="pages.checkout.address" /></label>
                <input id="checkout-address" autoComplete="street-address" value={form.address} onChange={(e) => set("address")(e.target.value)} className={FIELD} /></div>
              <div className="sm:col-span-2"><label htmlFor="checkout-city" className={LABEL}><T k="pages.checkout.city" /></label>
                <input id="checkout-city" autoComplete="address-level1" value={form.city} onChange={(e) => set("city")(e.target.value)} className={FIELD} /></div>
              <div className="sm:col-span-2"><label htmlFor="checkout-note" className={LABEL}><T k="pages.checkout.note" /></label>
                <textarea id="checkout-note" rows={2} value={form.note} onChange={(e) => set("note")(e.target.value)} className={FIELD} /></div>
            </div>
          </fieldset>
          <Rule />
          <fieldset>
            <Step n={2} k="pages.checkout.step_delivery" />
            <div className="border-t border-line">
              <OnlyRow testid="delivery-standard" icon="shipping" title={`${words.deliveryTitle} · ${standard?.days ?? ""}`} note={standard?.note} aside={words.free} />
            </div>
          </fieldset>
          <Rule />
          <fieldset>
            <Step n={3} k="pages.checkout.step_payment" />
            <div className="border-t border-line">
              {method && <OnlyRow testid="pay-vnpay" icon={(method.icon ?? "card") as IconName} title={method.name} note={method.note} />}
            </div>
          </fieldset>
        </div>

        <aside className="lg:col-span-5 lg:pl-8">
          <T k="pages.checkout.summary" as="h2" className="text-lg text-ink" />
          <ul className="mt-5 border-t border-line">
            {lines.map((l) => (
              <li key={lineKey(l)} data-record={`products:${l.id}`} className="flex gap-4 border-b border-line py-4">
                <div className="w-20 shrink-0"><ProductImage src={l.image} alt={l.name} aspect="1/1" /></div>
                <div className="flex min-w-0 flex-1 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink">{l.name}</p>
                    <p className="mt-1 text-sm text-ink-soft">{fill(words.size, { variant: l.variant ?? "", qty: l.qty })}</p>
                  </div>
                  <p className="shrink-0 text-sm tabular-nums text-ink">{formatPrice(l.price * l.qty)}</p>
                </div>
              </li>
            ))}
          </ul>
          <dl className="text-sm">
            <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.subtotal" /></dt><dd className="tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
            <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.delivery" /></dt><dd className="tabular-nums text-ink">{words.free}</dd></div>
            <div className="flex items-baseline justify-between py-4"><dt className="text-ink"><T k="cart.total" /></dt><dd data-testid="checkout-total" className="text-2xl tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
          </dl>
          <p className="flex items-start gap-2 border-t border-line pt-4 text-sm leading-relaxed text-ink-soft">
            <Icon name="info" size={16} className="shrink-0 text-ink" />
            {fill(words.nextOnline, { method: method?.name ?? "" })}
          </p>
          {errors.length > 0 && (
            <ul role="alert" className="mt-4 flex flex-col gap-1 text-xs text-danger">
              {errors.map((er) => <li key={er} className="flex items-center gap-2"><Icon name="alert" size={14} className="shrink-0" />{er}</li>)}
            </ul>
          )}
          <button type="submit" data-testid="place-order" disabled={paying}
            className="mt-5 flex w-full items-center justify-center gap-2 rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85 disabled:opacity-60">
            <Icon name="lock" size={16} />{paying ? words.paying : <T k="pages.checkout.place" />}
          </button>
          <T k="pages.checkout.terms" as="p" className="mt-3 text-sm leading-relaxed text-ink-soft" />
        </aside>
      </form>
    </>
  );
}
