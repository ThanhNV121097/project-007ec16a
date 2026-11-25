import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { clearCart, lineKey, useCart, type CartLine } from "../lib/cart";
import { useDemo } from "../lib/demo";
import { formatPrice } from "../lib/format";
import { placeOrder } from "../lib/mock/orders";
import { digitsOf, fill, type DeliveryOption, type PaymentMethod, type Product, type ShopTerms } from "../lib/shop";
import Icon, { type IconName } from "../components/Icon";
import EmptyBag from "../components/EmptyBag";
import ProductImage from "../components/ProductImage";
import Rule from "../components/Rule";

const FIELD = "w-full appearance-none rounded-sm border border-field bg-ground px-3 py-3 text-base text-ink outline-none placeholder:text-ink-soft focus:border-ink sm:text-sm";
const LABEL = "mb-1.5 block text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft";

function Step({ n, k }: { n: number; k: string }) {
  return (
    <legend className="mb-5 flex items-baseline gap-3 text-lg text-ink">
      <span className="text-sm tabular-nums text-ink-soft">{String(n).padStart(2, "0")}</span><T k={k} />
    </legend>
  );
}

/**
 * CHECKOUT, ON ONE PAGE: who and where, how fast, how to pay — three
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
  const deliveries = useCollection<DeliveryOption>("delivery", { pageSize: 10 }).items;
  const payments = useCollection<PaymentMethod>("payments", { pageSize: 10 }).items;
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const words = {
    free: useContent("pages.cart.free"),
    nextOnline: useContent("pages.checkout.next_online"),
    nextCod: useContent("pages.checkout.next_cod"),
    paying: useContent("pages.checkout.paying"),
    missing: useContent("pages.checkout.missing"),
    phoneBad: useContent("pages.checkout.phone_bad"),
    size: useContent("pages.checkout.line"),
  };
  const [delivery, setDelivery] = useState("standard");
  const [payment, setPayment] = useState("cod");
  const [form, setForm] = useState({ name: "", phone: "", address: "", city: "", note: "" });
  const [errors, setErrors] = useState<string[]>([]);
  const [paying, setPaying] = useState(false);

  const example: CartLine[] = products.slice(1, 3).map((p) => ({
    id: p.id, name: p.name, price: p.price, image: p.image, variant: `EU ${(p.sizes ?? [])[3] ?? ""}`, qty: 1,
  }));
  const showingExample = demo && real.length === 0;
  const lines = showingExample ? example : real;
  const subtotal = lines.reduce((s, l) => s + l.price * l.qty, 0);
  const option = deliveries.find((d) => d.id === delivery);
  const feeOf = (d: DeliveryOption) => (d.id === "standard" && terms && subtotal >= terms.free_delivery_from ? 0 : d.fee);
  const fee = option ? feeOf(option) : 0;
  const method = payments.find((m) => m.id === payment);
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
        lines, delivery, delivery_fee: fee, delivery_days: delivery === "express" ? 1 : 4,
        payment, online: Boolean(method?.online), customer: form.name.trim(), phone: form.phone.trim(),
        address: form.address.trim(), city: form.city.trim(), note: form.note.trim(),
      });
      if (!showingExample) clearCart();
      navigate(`/orders/${order.id}?placed=1`);
    }, method?.online ? 900 : 300);
  };

  return (
    <>
      <section className="flex flex-col gap-3 pb-8 pt-8 sm:flex-row sm:items-end sm:justify-between lg:pt-10">
        <T k="pages.checkout.heading" as="h1" className="text-[length:var(--size-h1)] leading-[1.02] text-ink" />
        <p className="flex items-center gap-2 pb-2 text-xs text-ink-soft"><Icon name="secure" size={16} className="shrink-0 text-ink" /><T k="pages.checkout.secure" /></p>
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
            <div className="border-t border-line" role="radiogroup">
              {deliveries.map((d) => {
                const on = d.id === delivery;
                const f = feeOf(d);
                return (
                  <label key={d.id} data-testid={`delivery-${d.id}`} className="flex cursor-pointer gap-4 border-b border-line py-4">
                    <input type="radio" name="delivery" value={d.id} checked={on} onChange={() => setDelivery(d.id)} className="sr-only" />
                    <span className={`mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border ${on ? "border-ink" : "border-field"}`}>{on && <span className="size-2 rounded-full bg-ink" />}</span>
                    <Icon name={d.id === "express" ? "fast" : "shipping"} size={18} className="mt-0.5 shrink-0 text-ink" />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm text-ink">{d.name} · {d.days}</span>
                      <span className="mt-1 block text-xs leading-relaxed text-ink-soft">{d.note}</span>
                    </span>
                    <span className="shrink-0 text-sm tabular-nums text-ink">{f === 0 ? words.free : formatPrice(f)}</span>
                  </label>
                );
              })}
            </div>
          </fieldset>
          <Rule />
          <fieldset>
            <Step n={3} k="pages.checkout.step_payment" />
            <div className="border-t border-line" role="radiogroup">
              {payments.map((m) => {
                const on = m.id === payment;
                return (
                  <label key={m.id} data-testid={`pay-${m.id}`} className="flex cursor-pointer gap-4 border-b border-line py-4">
                    <input type="radio" name="payment" value={m.id} checked={on} onChange={() => setPayment(m.id)} className="sr-only" />
                    <span className={`mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border ${on ? "border-ink" : "border-field"}`}>{on && <span className="size-2 rounded-full bg-ink" />}</span>
                    <Icon name={(m.icon ?? "card") as IconName} size={18} className="mt-0.5 shrink-0 text-ink" />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm text-ink">{m.name}</span>
                      <span className="mt-1 block text-xs leading-relaxed text-ink-soft">{m.note}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          </fieldset>
        </div>

        <aside className="lg:col-span-5 lg:pl-8">
          <T k="pages.checkout.summary" as="h2" className="text-lg text-ink" />
          <ul className="mt-5 border-t border-line">
            {lines.map((l) => (
              <li key={lineKey(l)} className="flex gap-4 border-b border-line py-4">
                <div className="w-20 shrink-0"><ProductImage src={l.image} alt={l.name} aspect="1/1" /></div>
                <div className="flex min-w-0 flex-1 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink">{l.name}</p>
                    <p className="mt-1 text-xs text-ink-soft">{fill(words.size, { variant: l.variant ?? "", qty: l.qty })}</p>
                  </div>
                  <p className="shrink-0 text-sm tabular-nums text-ink">{formatPrice(l.price * l.qty)}</p>
                </div>
              </li>
            ))}
          </ul>
          <dl className="text-sm">
            <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.subtotal" /></dt><dd className="tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
            <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.delivery" /></dt><dd className="tabular-nums text-ink">{fee === 0 ? words.free : formatPrice(fee)}</dd></div>
            <div className="flex items-baseline justify-between py-4"><dt className="text-ink"><T k="cart.total" /></dt><dd data-testid="checkout-total" className="text-2xl tabular-nums text-ink">{formatPrice(subtotal + fee)}</dd></div>
          </dl>
          <p className="flex items-start gap-2 border-t border-line pt-4 text-xs leading-relaxed text-ink-soft">
            <Icon name="info" size={16} className="shrink-0 text-ink" />
            {method?.online ? fill(words.nextOnline, { method: method.name }) : words.nextCod}
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
          <T k="pages.checkout.terms" as="p" className="mt-3 text-xs leading-relaxed text-ink-soft" />
        </aside>
      </form>
    </>
  );
}
