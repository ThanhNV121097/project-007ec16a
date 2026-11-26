import { describe, expect, it } from "vitest";
import {
  addOrder,
  customerSignIn,
  customerSignOut,
  customerSignUp,
  getAccount,
  listOrders,
  saveAccount,
} from "../lib/customer-accounts-sign-up-sign-in-my-orders";

/**
 * The story's client against its mock: the outcomes a screen branches on.
 * The mock keeps its state in memory, so the cases run in this order. The
 * backend step deletes the mock and this file with it; src/test/account-rules
 * stays.
 */
describe("the customer-account client on its mock", () => {
  it("refuses a wrong password and an unknown email in the same way", async () => {
    expect(await customerSignIn("lan@example.com", "wrong-password")).toEqual({ kind: "unauthenticated" });
    expect(await customerSignIn("nobody@example.com", "lanpass123")).toEqual({ kind: "unauthenticated" });
  });

  it("tells an unreachable shop and a lock from a refusal", async () => {
    expect(await customerSignIn("down@example.com", "lanpass123")).toEqual({ kind: "unreachable" });
    expect(await customerSignIn("limit@example.com", "lanpass123")).toEqual({ kind: "limited" });
    expect(await customerSignUp({ name: "A", email: "limit2@example.com", phone: "0912345678", password: "abcdefgh" })).toEqual({ kind: "limited" });
  });

  it("signs a customer in and lists two orders with no private field", async () => {
    const out = await customerSignIn("LAN@example.com", "lanpass123");
    expect(out.kind).toBe("ok");
    const orders = await listOrders(1);
    expect(orders.kind).toBe("ok");
    if (orders.kind !== "ok") return;
    expect(orders.value.total).toBe(2);
    expect(Object.keys(orders.value.items[0]).sort()).toEqual(["id", "number", "placed_at", "status", "total"]);
    expect(orders.value.items[0].total).toBe(6180000);
  });

  it("refuses saved details the shop dislikes and keeps good ones", async () => {
    const refused = await saveAccount({ name: "Lan Nguyen", phone: "0900 000 000", address: "", city: "" });
    expect(refused).toEqual({ kind: "invalid", errors: { phone: "Check the phone number" } });
    const saved = await saveAccount({ name: "Lan Nguyen", phone: "0912 345 678", address: "12 Example Street, Ward 1", city: "Da Nang" });
    expect(saved.kind).toBe("ok");
    const read = await getAccount();
    expect(read.kind === "ok" && read.value.city).toBe("Da Nang");
  });

  it("adds a guest order once however it is typed, and counts wrong tries", async () => {
    expect((await addOrder("doi 24818", "+84912345678")).kind).toBe("ok");
    expect((await addOrder("DOI-24818", "0912 345 678")).kind).toBe("ok");
    const orders = await listOrders(1);
    expect(orders.kind === "ok" && orders.value.total).toBe(3);
    expect(await addOrder("24801", "0912345678")).toEqual({ kind: "unreachable" });
    for (let i = 0; i < 5; i += 1) expect(await addOrder("24999", "0912345678")).toEqual({ kind: "notfound" });
    expect(await addOrder("doi 24818", "+84912345678")).toEqual({ kind: "limited" });
  });

  it("shows twelve orders ten at a time", async () => {
    await customerSignIn("many@example.com", "anything");
    const first = await listOrders(1);
    const second = await listOrders(2);
    expect(first.kind === "ok" && first.value.items.length).toBe(10);
    expect(second.kind === "ok" && second.value.items.length).toBe(2);
    expect(first.kind === "ok" && first.value.total).toBe(12);
  });

  it("refuses a team session on the account endpoints", async () => {
    await customerSignIn("minh@example.com", "minhpass123");
    expect(await getAccount()).toEqual({ kind: "forbidden" });
    expect(await listOrders(1)).toEqual({ kind: "forbidden" });
  });

  it("refuses an email held by any account, in any case, and signs a new customer up", async () => {
    const taken = await customerSignUp({ name: "Other", email: "LAN@Example.com", phone: "0912345678", password: "abcdefgh" });
    expect(taken.kind).toBe("conflict");
    const fresh = await customerSignUp({ name: "Bao Tran", email: "Bao@example.com", phone: "0987 654 321", password: "baopass123" });
    expect(fresh).toEqual({ kind: "ok", value: { name: "Bao Tran", email: "bao@example.com", role: "customer" } });
    const account = await getAccount();
    expect(account.kind === "ok" && account.value.address).toBe("");
    const orders = await listOrders(1);
    expect(orders.kind === "ok" && orders.value.items).toEqual([]);
  });

  it("opens nothing after signing out", async () => {
    await customerSignOut();
    expect(await getAccount()).toEqual({ kind: "unauthenticated" });
    expect(await listOrders(1)).toEqual({ kind: "unauthenticated" });
  });
});
