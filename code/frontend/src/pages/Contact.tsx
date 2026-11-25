import { Link, useSearchParams } from "react-router-dom";
import { T, useContent, useList } from "../editable";
import { usePageMeta } from "../seo";
import { useRecord } from "../data";
import { fill, type ShopTerms } from "../lib/shop";
import HeadPicture from "../components/HeadPicture";
import ContactForm from "../components/ContactForm";
import Icon from "../components/Icon";
import SpecList from "../components/SpecList";

/**
 * ASK THE SHOP: the routes a customer has, beside the form. At the left the
 * questions already answered elsewhere, as hairline rows with arrows, and
 * whatever ways to reach the shop the owner has given (none are invented);
 * at the right the form, which reaches the owner in Agent Team. A shoe's
 * "Ask about this pair" lands here with the pair named.
 */
export default function Contact() {
  usePageMeta("pages.contact.title", "pages.contact.description");
  const [params] = useSearchParams();
  const about = params.get("about");
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const quick = useList<{ label: string; href: string }>("pages.contact.quick");
  const lead = fill(useContent("pages.contact.lead"), { reply: terms?.reply_time ?? "" });
  const phone = useContent("contact.phone");
  const zalo = useContent("contact.zalo");
  const email = useContent("contact.email");
  const address = useContent("contact.address");
  const hours = useContent("contact.hours");
  const headAlt = useContent("pages.contact.head_alt");
  return (
    <>
      <section className="grid gap-x-8 gap-y-10 pb-14 pt-8 lg:grid-cols-12 lg:pt-10">
        <div className="flex flex-col justify-end lg:col-span-6">
          <h1 data-testid="contact-heading" className="max-w-[12ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">
            <T k="contact.heading" />
          </h1>
          {about && (
            <p className="mt-5 text-sm text-ink-soft"><T k="pages.contact.about" />{" "}<span className="text-ink">{about}</span></p>
          )}
          <p className="mt-5 max-w-[40ch] text-sm leading-relaxed text-ink-soft">{lead}</p>
        </div>
        <HeadPicture src="/images/about-stockroom.jpg" alt={headAlt} className="lg:col-span-6 lg:col-start-7" />
        <div className="lg:col-span-5">
          <div>
            <SpecList items={[
              [<span key="p" className="inline-flex items-center gap-2"><Icon name="phone" size={16} /><T k="pages.contact.phone" /></span>, phone && <a href={`tel:${phone}`} className="underline underline-offset-4">{phone}</a>],
              [<span key="z" className="inline-flex items-center gap-2"><Icon name="chat" size={16} /><T k="pages.contact.zalo" /></span>, zalo],
              [<span key="e" className="inline-flex items-center gap-2"><Icon name="mail" size={16} /><T k="pages.contact.email" /></span>, email && <a href={`mailto:${email}`} className="underline underline-offset-4">{email}</a>],
              [<span key="a" className="inline-flex items-center gap-2"><Icon name="address" size={16} /><T k="pages.contact.address" /></span>, address],
              [<span key="h" className="inline-flex items-center gap-2"><Icon name="hours" size={16} /><T k="pages.contact.hours" /></span>, hours],
            ]} />
          </div>
          <T k="pages.contact.quick_heading" as="h2" className="mt-10 text-lg text-ink" />
          <ul className="mt-4 border-t border-line">
            {quick.map((q, i) => (
              <li key={q.href} className="border-b border-line">
                <Link to={q.href} className="group flex items-center justify-between gap-4 py-4 text-sm text-ink">
                  <T k={`pages.contact.quick.${i}.label`} className="underline-offset-4 group-hover:underline" />
                  <Icon name="arrow" size={16} className="shrink-0 transition-transform group-hover:translate-x-1" />
                </Link>
              </li>
            ))}
          </ul>
        </div>
        <div className="lg:col-span-6 lg:col-start-7">
          <T k="pages.contact.form_heading" as="h2" className="mb-6 text-lg text-ink" />
          <ContactForm />
        </div>
      </section>
    </>
  );
}
