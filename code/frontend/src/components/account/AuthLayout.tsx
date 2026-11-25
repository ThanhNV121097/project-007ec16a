import { type ReactNode } from "react";
import Rule from "../Rule";
import { usePageTitle } from "./hooks";

/**
 * The frame of sign-up and sign-in inside the customer shell: the heading at
 * display size (with an optional line under it) and the form beside it from
 * 1024px, one column below, then a hairline.
 */
export default function AuthLayout({ title, lead, children }: { title: string; lead?: string; children: ReactNode }) {
  usePageTitle(title);
  return (
    <>
      <section className="grid gap-x-8 gap-y-8 pb-12 pt-8 lg:grid-cols-12 lg:pt-10">
        <div className="min-w-0 lg:col-span-6">
          <h1 className="wrap-anywhere text-[length:var(--size-h1)] leading-[1.02] text-ink">{title}</h1>
          {lead && <p className="mt-5 max-w-[42ch] text-sm leading-relaxed text-ink-soft">{lead}</p>}
        </div>
        <div className="min-w-0 max-w-md lg:col-span-5 lg:col-start-8">{children}</div>
      </section>
      <Rule />
    </>
  );
}
