import { useMemo } from "react";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { heldOrders } from "../lib/track-an-order-by-number-and-phone";
import { ORDER_STEPS } from "../lib/shop";
import HeadPicture from "../components/HeadPicture";
import Disclosure from "../components/Disclosure";
import FaqRows from "../components/FaqRows";
import HeldOrders from "../components/HeldOrders";
import OutlinePill from "../components/OutlinePill";
import PromiseRow from "../components/PromiseRow";
import TrackForm from "../components/TrackForm";

/**
 * TRACK AN ORDER: the order number and the phone it was placed with, and
 * the answer is the order's own page. Beside the form, what each of the
 * five steps means — the question a buyer asks while waiting — and the
 * orders this browser already holds.
 */
export default function Track() {
  usePageMeta("pages.track.title", "pages.track.description");
  const headAlt = useContent("pages.track.head_alt");
  const steps = useContent<Record<string, string>>("pages.track.steps");
  const labels = useContent<Record<string, string>>("orders.steps");
  const held = useMemo(heldOrders, []);

  return (
    <>
      <section className="grid gap-x-8 gap-y-10 pb-12 pt-8 lg:grid-cols-12 lg:pt-10">
        <div className="flex flex-col justify-end lg:col-span-6">
          <T k="pages.track.heading" as="h1" className="max-w-[10ch] text-[length:var(--size-h1)] leading-[1.02] text-ink" />
          <T k="pages.track.lead" as="p" className="mt-5 max-w-[42ch] text-sm leading-relaxed text-ink-soft" />
        </div>
        <HeadPicture src="/images/head-track.jpg" alt={headAlt} className="lg:col-span-5 lg:col-start-8" />
        <div className="lg:col-span-6">
          <TrackForm />
        </div>
        <div className="lg:col-span-5 lg:col-start-8">
          <T k="pages.track.steps_heading" as="h2" className="mb-5 text-lg text-ink" />
          <Disclosure rows={ORDER_STEPS.map((s) => ({ label: labels?.[s] ?? s, content: <p className="max-w-[48ch] leading-relaxed">{steps?.[s]}</p> }))} />
          <div className="mt-10"><PromiseRow compact /></div>
          <HeldOrders orders={held} />
        </div>
      </section>
      <section className="flex flex-col gap-8 border-t border-line py-12 lg:flex-row lg:gap-12">
        <div className="flex flex-col items-start gap-5 lg:w-1/3">
          <T k="pages.track.faq_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
          <T k="pages.track.help" as="p" className="max-w-[36ch] text-sm leading-relaxed text-ink-soft" />
          <OutlinePill to="/contact"><T k="pages.track.contact_cta" /></OutlinePill>
        </div>
        <div className="lg:flex-1">
          <FaqRows topic="orders" />
          <FaqRows topic="shipping" />
        </div>
      </section>
    </>
  );
}
