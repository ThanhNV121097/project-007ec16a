import { type ReactNode } from "react";
import { useContent } from "../../editable";
import HeadPicture from "../HeadPicture";
import Rule from "../Rule";
import { usePageTitle } from "./hooks";

/** The photographs the site already has, with the sentences the owner already edits for them. */
const ART = {
  track: { src: "/images/head-track.jpg", alt: "pages.track.head_alt" },
  checkout: { src: "/images/head-checkout.jpg", alt: "pages.checkout.head_alt" },
} as const;

/**
 * The frame of sign-up and sign-in inside the customer shell, built like every
 * page under the home page: the title at the left and one photograph that runs
 * off the right edge from 1024px, below the form on a phone, where the page is
 * one column. From 1024px the section is as tall as the first screen, with the
 * title at its top and the form at its foot, so a short form does not leave a
 * corner of the screen empty.
 */
export default function AuthLayout({ title, lead, art, children }: { title: string; lead?: string; art: keyof typeof ART; children: ReactNode }) {
  usePageTitle(title);
  const alt = useContent(ART[art].alt);
  return (
    <>
      <section className="grid gap-x-8 gap-y-10 pb-12 pt-8 lg:min-h-[calc(100svh-var(--header-h)-5rem)] lg:grid-cols-12 lg:pt-10">
        <div className="flex min-w-0 flex-col justify-between gap-8 lg:col-span-6">
          <div>
            <h1 className="wrap-anywhere text-[length:var(--size-h1)] leading-[1.02] text-ink">{title}</h1>
            {lead && <p className="mt-5 max-w-[42ch] text-sm leading-relaxed text-ink-soft">{lead}</p>}
          </div>
          <div className="max-w-md">{children}</div>
        </div>
        <HeadPicture src={ART[art].src} alt={alt} className="lg:col-span-5 lg:col-start-8" />
      </section>
      <Rule />
    </>
  );
}
