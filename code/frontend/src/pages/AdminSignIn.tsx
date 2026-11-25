import { T, useContent, useList } from "../editable";
import { SignInForm } from "../components/admin/SignInPage";
import Icon, { type IconName } from "../components/Icon";
import Logo from "../components/Logo";
import { Picture } from "../picture";

/** What each line of "what is inside" is drawn beside, in order. */
const INSIDE_ICONS: readonly IconName[] = ["package", "tag", "alert"];

/**
 * THE WAY INTO THE SHOP'S OWN SIDE, as a split screen: the stockroom down the
 * left, running off the edge, and at the right the name, what the area holds
 * and the form. It is also where the dealer lands after signing out of any
 * page of the area, so it says what is behind it rather than standing as a
 * bare card in the middle of an empty screen.
 */
export default function AdminSignIn() {
  const inside = useList<{ text: string }>("pages.admin.signin.inside");
  return (
    <div data-dashboard="true" className="grid min-h-svh bg-ground font-body text-ink lg:grid-cols-2">
      <div className="relative h-56 overflow-hidden bg-surface sm:h-72 lg:h-auto">
        <Picture src="/images/about-stockroom.jpg" alt={useContent("pages.admin.signin.image_alt")} width={1280} height={714}
          loading="eager" className="absolute inset-0 h-full w-full object-cover" />
      </div>
      <div className="flex items-center justify-center px-6 py-10 sm:px-12">
        <div className="flex w-full max-w-sm flex-col gap-6">
          <Logo />
          <div>
            <T k="pages.admin.signin.title" as="h1" className="text-2xl font-medium text-ink" />
            <T k="pages.admin.signin.description" as="p" className="mt-2 text-sm leading-relaxed text-ink-soft" />
          </div>
          <ul className="flex flex-col gap-3 border-y border-line py-4">
            {inside.map((line, i) => (
              <li key={i} className="flex gap-3 text-sm text-ink">
                <Icon name={INSIDE_ICONS[i] ?? "check"} size={18} className="mt-0.5 text-ink-soft" />
                <T k={`pages.admin.signin.inside.${i}.text`} className="leading-relaxed">{line.text}</T>
              </li>
            ))}
          </ul>
          <SignInForm to="/admin" note={useContent("pages.admin.signin.note")} />
        </div>
      </div>
    </div>
  );
}
