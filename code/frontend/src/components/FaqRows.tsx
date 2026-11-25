import { useCollection } from "../data";
import type { Faq } from "../lib/shop";
import Disclosure from "./Disclosure";

/** The questions customers ask about one page's subject, as disclosure rows. */
export default function FaqRows({ topic, className = "" }: { topic: string; className?: string }) {
  const { items } = useCollection<Faq>("faqs", { where: { topic }, pageSize: 20 });
  return (
    <div className={className}>
      <Disclosure rows={items.map((f) => ({ label: f.question, content: <p className="max-w-[60ch] leading-relaxed">{f.answer}</p> }))} />
    </div>
  );
}
