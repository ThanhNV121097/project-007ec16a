import { useSyncExternalStore } from "react";

/**
 * Whether the bag's drawer is open. A store rather than a prop because the
 * header draws the drawer and the shoe's page is what opens it, the moment a
 * pair goes in.
 */
let open = false;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

export function openBag(): void {
  open = true;
  emit();
}

export function closeBag(): void {
  open = false;
  emit();
}

export function useBagOpen(): boolean {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => {
        listeners.delete(l);
      };
    },
    () => open,
    () => false,
  );
}
