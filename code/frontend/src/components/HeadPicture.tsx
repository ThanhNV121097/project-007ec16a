import { Picture } from "../picture";
import Bleed from "./Bleed";

/**
 * THE PICTURE BESIDE A PAGE'S TITLE. Every page under the home page opens the
 * way the home page does, smaller: its title at the left and one photograph
 * that runs off the right edge of the screen, as tall as the title beside it
 * (at least 14rem). On a phone it is a strip under the title, edge to edge.
 *
 * It is a grid column of the page's first section: give it the columns in
 * className (`lg:col-span-6 lg:col-start-7`).
 */
export default function HeadPicture({ src, alt, className = "" }: { src: string; alt: string; className?: string }) {
  return (
    <Bleed side="right" className={`relative aspect-[16/9] overflow-hidden bg-surface lg:aspect-auto lg:min-h-56 ${className}`}>
      <Picture src={src} alt={alt} width={1280} height={853} loading="eager" className="absolute inset-0 h-full w-full object-cover" />
    </Bleed>
  );
}
