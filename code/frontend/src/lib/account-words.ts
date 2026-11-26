import type { OrderStatus } from "./shop";

/**
 * EVERY WORD THIS STORY ADDS, in one place.
 *
 * The owner's Studio edits src/content.json, and that file is frozen while
 * stories are built side by side (overview.md section 5), so the sentences of
 * the customer-account screens live here and are drawn by their components.
 * They are the SRS's own words (docs/accounts/SRS.md) and the dealer sign-in's
 * for the two blank-field sentences. The words the site already owns (the
 * header's links, "Keep shopping", "We can't find that order.") are still read
 * from content.json by the components that draw them.
 */
export const W = {
  unreachable: "We cannot reach the server. Please try again.",
  rules: {
    name: "Fill in your name",
    email: "Enter a valid email address",
    phone: "Check the phone number",
    passwordShort: "Use at least 8 characters",
    max100: "Use at most 100 characters",
    max200: "Use at most 200 characters",
  },
  labels: { name: "Name", email: "Email", phone: "Phone", password: "Password" },
  signUp: {
    title: "Create your account",
    lead: "Optional — you can always buy without one.",
    submit: "Create account",
    busy: "Creating your account…",
    taken: "An account with this email already exists. Sign in instead.",
    limited: "Too many tries. Please wait and try again.",
    have: "Already have an account?",
    haveLink: "Sign in",
  },
  signIn: {
    title: "Sign in",
    submit: "Sign in",
    busy: "Signing in…",
    needEmail: "An email address is needed",
    needPassword: "A password is needed",
    wrong: "Wrong email or password.",
    limited: "Too many tries. Please wait 15 minutes.",
    fresh: "New here?",
    freshLink: "Create an account",
    guest: "Buying as a guest needs no account.",
  },
  entry: { signIn: "Sign in" },
  account: {
    title: "My account",
    signOut: "Sign out",
  },
  orders: {
    title: "My orders",
    loading: "Loading your orders…",
    empty: "You have no orders yet.",
    more: "Show more",
    failed: "We could not load your orders.",
    retry: "Try again",
  },
  details: {
    title: "Saved delivery details",
    name: "Name",
    phone: "Phone",
    address: "Street address, ward, district",
    city: "City or province",
    save: "Save details",
    saving: "Saving…",
    saved: "Saved.",
    failed: "We could not save. Please try again.",
  },
  add: {
    title: "Add an order",
    number: "Order number",
    placeholder: "DOI-24817",
    phone: "Phone",
    submit: "Add order",
    added: "Added.",
    failed: "We could not add that order. Please try again.",
    // The sentence is cut where "message us" is a link to /contact.
    notFound: ["No order matches that number and phone. Check both, or ", "message us", "."],
    limited: ["Too many tries. Please wait 15 minutes, or ", "message us", " with your order number."],
  },
} as const;

/** "Hello, Lan Nguyen". */
export const hello = (name: string): string => `Hello, ${name}`;

/** An order's status in the customer's words ("shipped" is "On its way"). */
export const STATUS_WORDS: Record<OrderStatus, string> = {
  placed: "Placed",
  confirmed: "Confirmed",
  packed: "Packed",
  shipped: "On its way",
  delivered: "Delivered",
  cancelled: "Cancelled",
};
