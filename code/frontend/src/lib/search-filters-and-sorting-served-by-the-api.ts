import { api } from "./api";
import type { Product } from "./shop";
import { toParams, type Page, type ProductFilters, type ProductsRequest } from "./shop-request";

/**
 * THE SHOP'S TWO READS OF THE API, over the project's one client
 * (src/lib/api.ts). A failed request rejects with the client's ApiError — a
 * 5xx, an unreachable server and an offline browser alike — and
 * src/lib/shop-list.ts, the only importer of this file, turns the rejection
 * into the hook's `error`.
 */

/** `GET /v1/products?status=live&limit=12&page=n&…`: a page of live shoes, with `total` every match. */
export function listProducts(req: ProductsRequest): Promise<Page<Product>> {
  return api.get<Page<Product>>("products", toParams(req));
}

/** `GET /v1/product-filters`: every colour and EU size a live shoe has, whatever is filtered. */
export function getProductFilters(): Promise<ProductFilters> {
  return api.get<ProductFilters>("product-filters");
}
