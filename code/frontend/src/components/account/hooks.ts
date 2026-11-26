import { useEffect, type RefObject } from "react";
import { useContent } from "../../editable";

/** The tab's title for a screen of this story: "Sign in — Đôi". */
export function usePageTitle(title: string): void {
  const name = useContent("site.name");
  useEffect(() => {
    document.title = name ? `${title} — ${name}` : title;
  }, [title, name]);
}

/**
 * When a form is refused, the first field at fault takes focus: a keyboard or
 * screen-reader user lands on the sentence that says what to fix. Runs only
 * when a new set of problems is set, never while typing.
 */
export function useFocusFirstInvalid(root: RefObject<HTMLElement | null>, problems: Record<string, string>): void {
  useEffect(() => {
    root.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
  }, [root, problems]);
}
