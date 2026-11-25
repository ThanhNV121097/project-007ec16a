/**
 * What this shop's records are, as the pages read them. The fields are the
 * ones src/data/schema.json declares; change both together.
 */
export interface Product {
  id: string;
  name: string;
  category: string;
  price: number;
  stock?: number;
  status?: string;
  image?: string;
  image_back?: string;
  image_detail?: string;
  colours?: string[];
  sizes?: string[];
  material?: string;
  care?: string;
  description?: string;
  featured?: boolean;
}

export interface Kind {
  id: string;
  name: string;
  blurb?: string;
}

/** Every picture a product has, the main one first. */
export function picturesOf(p: Product): string[] {
  return [p.image, p.image_back, p.image_detail].filter((s): s is string => Boolean(s));
}
