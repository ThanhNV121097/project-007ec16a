import { useSearchParams } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import ContactForm from "../components/ContactForm";
import Rule from "../components/Rule";
import SpecList from "../components/SpecList";

/**
 * How to reach the shop, and the form that reaches it.
 *
 * A product's "Ask about this" lands here with the product named, so the
 * visitor is not asked to type again what they were just looking at. The
 * details at the left are drawn only where the owner has given them: a shop
 * with no street address shows no Address row.
 */
export default function Contact() {
  usePageMeta("pages.contact.title", "pages.contact.description");
  const [params] = useSearchParams();
  const about = params.get("about");
  const phone = useContent("contact.phone");
  const email = useContent("contact.email");
  const address = useContent("contact.address");
  const hours = useContent("contact.hours");
  return (
    <>
      <section className="py-10">
        <h1 data-testid="contact-heading" className="max-w-[12ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">
          <T k="contact.heading" />
        </h1>
        {about && (
          <p className="mt-6 text-sm text-ink-soft">
            <T k="pages.contact.about" />{" "}<span className="text-ink">{about}</span>
          </p>
        )}
      </section>
      <Rule />
      <section className="flex flex-col gap-12 py-12 lg:flex-row lg:gap-16">
        <div className="lg:w-5/12">
          <T k="pages.contact.lead" as="p" className="mb-8 max-w-[36ch] text-sm leading-relaxed text-ink-soft" />
          <SpecList items={[
            [<T key="p" k="pages.contact.phone" />, phone && <a href={`tel:${phone}`} className="underline underline-offset-4">{phone}</a>],
            [<T key="e" k="pages.contact.email" />, email && <a href={`mailto:${email}`} className="underline underline-offset-4">{email}</a>],
            [<T key="a" k="pages.contact.address" />, address],
            [<T key="h" k="pages.contact.hours" />, hours],
          ]} />
        </div>
        <div className="lg:w-7/12">
          <ContactForm className="max-w-[40rem]" />
        </div>
      </section>
    </>
  );
}
