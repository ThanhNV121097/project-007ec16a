/**
 * What this shop's records are, as the pages read them. The fields are the
 * ones src/data/schema.json declares; change both together.
 */
export interface Product {
  id: string;
  name: string;
  model?: string;
  line: string;
  category: string;
  price: number;
  status?: string;
  featured?: boolean;
  is_new?: boolean;
  description?: string;
  code?: string;
  colourway?: string;
  colour?: string;
  swatch?: string;
  upper?: string;
  lining?: string;
  sole?: string;
  fit?: string;
  care?: string;
  released?: string;
  sizes?: string[];
  sizes_low?: string[];
  sizes_out?: string[];
  stock?: number;
  rating?: number;
  review_count?: number;
  image?: string;
  image_back?: string;
  image_detail?: string;
}

export interface Line {
  id: string;
  name: string;
  blurb?: string;
  since?: number;
  fit_note?: string;
}

export interface Category {
  id: string;
  name: string;
  blurb?: string;
  image?: string;
}

export interface Review {
  id: string;
  product: string;
  name: string;
  rating: number;
  date: string;
  size_bought?: string;
  fit?: "small" | "true" | "large";
  body: string;
  status?: string;
}

export type OrderStatus = "placed" | "confirmed" | "packed" | "shipped" | "delivered" | "cancelled";

export interface Order {
  id: string;
  number: string;
  placed_at: string;
  status: OrderStatus;
  product: string;
  size: string;
  qty: number;
  total: number;
  payment: string;
  paid: boolean;
  delivery: string;
  carrier?: string;
  tracking?: string;
  eta?: string;
  customer: string;
  phone?: string;
  address?: string;
  city?: string;
}

export interface ShopTerms {
  id: string;
  free_delivery_from: number;
  standard_days: string;
  express_days: string;
  express_cities: string;
  dispatch_cutoff: string;
  exchange_days: number;
  refund_days: number;
  reply_time: string;
  rating: number;
  review_count: number;
}

export interface DeliveryOption {
  id: string;
  name: string;
  days: string;
  fee: number;
  note?: string;
}

export interface PaymentMethod {
  id: string;
  name: string;
  note?: string;
  online?: boolean;
  icon?: string;
}

export interface Faq {
  id: string;
  topic: string;
  question: string;
  answer: string;
}

export interface SizeRow {
  id: string;
  eu: string;
  uk: string;
  us_m: string;
  us_w: string;
  cm: number;
}

/** Every picture a shoe has, the side view first. */
export function picturesOf(p: Product): string[] {
  return [p.image, p.image_back, p.image_detail].filter((s): s is string => Boolean(s));
}

/** The sizes a shoe can be bought in today. */
export function sizesInStock(p: Product): string[] {
  const out = new Set(p.sizes_out ?? []);
  return (p.sizes ?? []).filter((s) => !out.has(s));
}

/** A size as an id or a test hook: "36 2/3" → "36-2-3". */
export function sizeKey(size: string): string {
  return size.trim().replace(/[\s/]+/g, "-");
}

/** The price bands the catalogue narrows by. */
export const PRICE_BANDS: readonly { id: string; from: number; to: number }[] = [
  { id: "under-3m", from: 0, to: 2999999 },
  { id: "3m-4m", from: 3000000, to: 3999999 },
  { id: "4m-plus", from: 4000000, to: Number.MAX_SAFE_INTEGER },
];

export function inBand(price: number, band: string): boolean {
  const b = PRICE_BANDS.find((x) => x.id === band);
  return b ? price >= b.from && price <= b.to : true;
}

/** The five steps an order walks, in order. */
export const ORDER_STEPS: readonly OrderStatus[] = ["placed", "confirmed", "packed", "shipped", "delivered"];

/** Puts values into a sentence from content.json: "{n}-day exchange". */
export function fill(sentence: string, values: Record<string, string | number>): string {
  return sentence.replace(/\{(\w+)\}/g, (all, key: string) => (key in values ? String(values[key]) : all));
}

/** Digits only, for matching a phone number however it was typed. */
export function digitsOf(s: string | undefined): string {
  return (s ?? "").replace(/\D/g, "");
}
