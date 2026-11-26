import type { Overview } from "../overview";

/**
 * UI-stage stand-in for GET /v1/admin/overview (services §3.25). Deleted by the
 * backend step, when src/lib/overview.ts reads the real endpoint.
 *
 * The sample is the SRS's answer at its fixed clock (25/11/2026): 13 orders,
 * 47.660.000 ₫, two shoes running low, five lines, five latest orders.
 */
export const sampleOverview: Overview = {
  month: { sales: 47660000, orders: 13 },
  waiting_orders: 2,
  pairs_on_shelf: 73,
  shoes: 3,
  running_low: [
    { id: "gazelle-indoor-solar-gold", name: "Gazelle Indoor Solar Gold", category: "men", image: "/images/ref/gazelle-indoor-solar-gold.jpg", stock: 5, sizes_low: ["40", "41"] },
    { id: "samba-og-kids-cloud-white", name: "Samba OG Kids Cloud White", category: "kids", image: "/images/ref/samba-og-kids-cloud-white.jpg", stock: 9, sizes_low: ["36"] },
  ],
  sales_by_day: [
    { date: "2026-11-12", sales: 2790000 }, { date: "2026-11-13", sales: 3590000 },
    { date: "2026-11-14", sales: 3090000 }, { date: "2026-11-15", sales: 3790000 },
    { date: "2026-11-16", sales: 0 }, { date: "2026-11-17", sales: 2290000 },
    { date: "2026-11-18", sales: 5090000 }, { date: "2026-11-19", sales: 3290000 },
    { date: "2026-11-20", sales: 0 }, { date: "2026-11-21", sales: 5580000 },
    { date: "2026-11-22", sales: 3590000 }, { date: "2026-11-23", sales: 5380000 },
    { date: "2026-11-24", sales: 5890000 }, { date: "2026-11-25", sales: 3290000 },
  ],
  pairs_by_line: [
    { line: "samba", name: "Samba", pairs: 8 },
    { line: "gazelle", name: "Gazelle", pairs: 2 },
    { line: "megaride", name: "Megaride", pairs: 2 },
    { line: "handball-spezial", name: "Handball Spezial", pairs: 1 },
    { line: "adizero", name: "Adizero", pairs: 1 },
  ],
  latest_orders: [
    { id: "doi-24816", number: "DOI-24816", customer: "Trang", product: "handball-spezial-w-earth-strata", product_name: "Handball Spezial Earth Strata", size: "37 1/3", status: "placed", placed_at: "2026-11-25T08:40:00+07:00", total: 3290000 },
    { id: "doi-24815", number: "DOI-24815", customer: "Long", product: "megaride-bones-halo-green", product_name: "Megaride Bones Halo Green", size: "43 1/3", status: "confirmed", placed_at: "2026-11-24T10:12:00+07:00", total: 5890000 },
    { id: "doi-24814", number: "DOI-24814", customer: "Hương", product: "samba-og-kids-cloud-white", product_name: "Samba OG Kids Cloud White", size: "37 1/3", status: "packed", placed_at: "2026-11-23T19:30:00+07:00", total: 2290000 },
    { id: "doi-24817", number: "DOI-24817", customer: "Hoàng Nam", product: "samba-og-w-crystal-white", product_name: "Samba OG Crystal White", size: "39 1/3", status: "shipped", placed_at: "2026-11-23T16:05:00+07:00", total: 3090000 },
    { id: "doi-24813", number: "DOI-24813", customer: "Tuấn", product: "gazelle-indoor-solar-gold", product_name: "Gazelle Indoor Solar Gold", size: "43 1/3", status: "shipped", placed_at: "2026-11-22T11:02:00+07:00", total: 3590000 },
  ],
};

/** A shop with nothing sold and nothing running low: every list empty, every day 0. */
export const emptyOverview: Overview = {
  month: { sales: 0, orders: 0 },
  waiting_orders: 0,
  pairs_on_shelf: 0,
  shoes: 0,
  running_low: [],
  sales_by_day: sampleOverview.sales_by_day.map((d) => ({ date: d.date, sales: 0 })),
  pairs_by_line: [],
  latest_orders: [],
};

/** The answer after a short wait, as the network would give it. */
export function readOverview(): Promise<Overview> {
  return new Promise((resolve) => {
    setTimeout(() => resolve(sampleOverview), 400);
  });
}
