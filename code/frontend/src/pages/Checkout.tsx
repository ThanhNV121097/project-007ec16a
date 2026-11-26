import { useRef, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { clearCart, useCart, type CartLine } from "../lib/cart";
import { useSavedDetailsFill } from "../lib/customer-accounts-sign-up-sign-in-my-orders";
import { useDemo } from "../lib/demo";
import {
  buildRequest, detailProblems, keyKeeper, payAddress, placeOrder, rememberOrder, useCurrentPrices, usePaymentOptions,
  type Details, type PlaceResult,
} from "../lib/orders";
import { chosenWay, waysOf } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import type { DeliveryOption, Product } from "../lib/shop";
import HeadPicture from "../components/HeadPicture";
import Icon from "../components/Icon";
import CheckoutRow from "../components/CheckoutRow";
import CheckoutSummary from "../components/CheckoutSummary";
import EmptyBag from "../components/EmptyBag";
import PaymentRows from "../components/PaymentRows";
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
 * CHECKOUT, ON ONE PAGE: who and where, delivery, how to pay — three
 * numbered groups because they are a sequence — with the pairs, their
 * pictures and the total kept in view beside them, and what happens after
 * "Place order" said before it is pressed. The shop reads its own prices and
 * sets the pairs aside; this page sends what was typed, the pairs and the way
 * to pay chosen among those the shop lists, never an amount (src/lib/orders.ts).
 */
export default function Checkout() {
  usePageMeta("pages.checkout.title", "pages.checkout.description");
  const navigate = useNavigate();
  const { lines: real } = useCart();
  const demo = useDemo();
  const products = useCollection<Product>("products", { where: { status: "live" }, pageSize: 3 }).items;
  const standard = useRecord<DeliveryOption>("delivery", "standard").item;
  const options = usePaymentOptions();
  const current = useCurrentPrices(real.map((l) => l.id));
  const words = {
    free: useContent("pages.cart.free"),
    deliveryTitle: useContent("pages.checkout.delivery_title"),
    missing: useContent("pages.checkout.missing"),
    phoneBad: useContent("pages.checkout.phone_bad"),
    failed: useContent("pages.checkout.failed"),
    offline: useContent("pages.checkout.offline"),
    unavailable: useContent("pages.checkout.unavailable"),
    headAlt: useContent("pages.checkout.head_alt"),
  };
  const [form, setForm] = useState<Details>({ name: "", phone: "", address: "", city: "", note: "" });
  useSavedDetailsFill(setForm);
  const [problems, setProblems] = useState<string[]>([]);
  const [placing, setPlacing] = useState(false);
  const [picked, setPicked] = useState("");
  const [closedWays, setClosedWays] = useState<string[]>([]);
  const [keyFor] = useState(keyKeeper);
  const sending = useRef(false);

  const example: CartLine[] = products.slice(1, 3).map((p) => ({
    id: p.id, name: p.name, price: p.price, image: p.image, variant: `EU ${(p.sizes ?? [])[3] ?? ""}`, qty: 1,
  }));
  const showingExample = demo && real.length === 0;
  // The shop's price today, not the one the bag saved: a shoe the shop does not
  // answer for keeps its saved price here and is refused by name when ordered.
  const lines = (showingExample ? example : real).map((l) => ({ ...l, price: showingExample ? l.price : current.get(l.id) ?? l.price }));
  const subtotal = lines.reduce((s, l) => s + l.price * l.qty, 0);
  // The ways the shop lists, in its order; one it refused to place an order with is closed.
  // The first open one is chosen until the visitor chooses another, and Place order is off
  // only when none is open (or the read of what it offers failed, and the page says so).
  const ways = waysOf(options.items, closedWays);
  const chosen = chosenWay(ways, picked);
  const nothing = !options.loading && ways.length === 0;
  const blocked = !chosen;
  const set = (k: keyof Details) => (v: string) => setForm((f) => ({ ...f, [k]: v }));

  if (lines.length === 0) return <EmptyBag heading="pages.checkout.empty" />;

  const stopped = (r: Extract<PlaceResult, { ok: false }>): string[] => {
    if (r.kind === "unavailable") return [];
    if (r.kind === "offline") return [words.offline];
    return r.problems.length > 0 ? r.problems : [words.failed];
  };

  const place = async (e: FormEvent) => {
    e.preventDefault();
    if (sending.current || !chosen) return;
    const wrong = detailProblems(form).map((p) => (p === "missing" ? words.missing : words.phoneBad));
    setProblems(wrong);
    if (wrong.length > 0) return;
    sending.current = true;
    setPlacing(true);
    const request = buildRequest(lines, form, chosen.id);
    const result = await placeOrder(request, keyFor(request));
    if (!result.ok) {
      setProblems(stopped(result));
      if (result.kind === "unavailable") setClosedWays((c) => [...c, chosen.id]);
      sending.current = false;
      setPlacing(false);
      return;
    }
    rememberOrder(result.placed);
    if (!showingExample) clearCart();
    const { payment_url: pay, order } = result.placed;
    const address = payAddress(pay);
    if (address) window.location.assign(address);
    else navigate(`/orders/${order.id}?placed=1`);
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
      <form onSubmit={(e) => void place(e)} noValidate className="grid gap-12 py-10 lg:grid-cols-12 lg:gap-8">
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
              <CheckoutRow testid="delivery-standard" icon="shipping" title={`${words.deliveryTitle} · ${standard?.days ?? ""}`} note={standard?.note} aside={words.free} />
            </div>
          </fieldset>
          <Rule />
          <fieldset>
            <Step n={3} k="pages.checkout.step_payment" />
            <PaymentRows options={ways} chosen={chosen?.id ?? ""} onChoose={setPicked} />
            {nothing && (
              <p role="alert" className="mt-3 flex items-center gap-2 text-xs text-danger">
                <Icon name="alert" size={14} className="shrink-0" />{words.unavailable}
              </p>
            )}
          </fieldset>
        </div>
        <CheckoutSummary lines={lines} subtotal={subtotal} method={(chosen ?? ways[0])?.name} problems={problems} placing={placing} blocked={blocked} />
      </form>
    </>
  );
}
