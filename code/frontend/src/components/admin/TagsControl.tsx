import { useState } from "react";
import Icon from "../Icon";
import { text } from "./fields";
import { invalid, type ControlProps } from "./control-props";
import { Field } from "./index";

/**
 * A list of short values — sizes, tags — as things you add and remove.
 * `normalise` says how typed text becomes one value; trimming by default.
 */
export default function TagsControl({ spec, value, error, onChange, normalise = (s) => s.trim() }: ControlProps & {
  normalise?: (typed: string) => string;
}) {
  const [draft, setDraft] = useState("");
  const tags = Array.isArray(value)
    ? value.map(String)
    : text(value).split(",").map((s) => s.trim()).filter(Boolean);
  const add = () => {
    const next = normalise(draft);
    if (next && !tags.includes(next)) onChange([...tags, next]);
    setDraft("");
  };
  return (
    <Field label={spec.label} hint={spec.hint ?? "Press Enter to add one."} group>
      <div className="flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-field bg-surface px-2 py-1.5 shadow-xs">
        {tags.map((tag) => (
          <span key={tag} className="inline-flex items-center gap-1 rounded-md bg-ground px-2 py-0.5 text-sm font-normal text-ink">
            {tag}
            <button type="button" aria-label={`Remove ${tag}`} className="text-ink-soft hover:text-ink"
              onClick={() => onChange(tags.filter((t) => t !== tag))}>
              <Icon name="close" size={12} />
            </button>
          </span>
        ))}
        <input value={draft} aria-label={`Add to ${spec.label}`} {...invalid(spec.key, error)}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={add}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === ",") { e.preventDefault(); add(); }
            if (e.key === "Backspace" && draft === "" && tags.length > 0) onChange(tags.slice(0, -1));
          }}
          className="min-w-20 flex-1 bg-transparent text-sm font-normal text-ink outline-none" />
      </div>
    </Field>
  );
}
