import { describe, expect, it } from "vitest";
import { checkDetails, checkSignIn, checkSignUp } from "../lib/account-rules";

const GOOD = { name: "Lan Nguyen", email: "lan@example.com", phone: "0912 345 678", password: "lanpass123" };

describe("sign-up rules (SC-19)", () => {
  it("names every empty field and nothing else", () => {
    expect(checkSignUp({ name: "", email: "", phone: "", password: "" })).toEqual({
      name: "Fill in your name",
      email: "Enter a valid email address",
      phone: "Check the phone number",
      password: "Use at least 8 characters",
    });
  });

  it("gives each wrong value its own sentence", () => {
    expect(checkSignUp({ ...GOOD, email: "lan@" })).toEqual({ email: "Enter a valid email address" });
    expect(checkSignUp({ ...GOOD, email: "lan@example" })).toEqual({ email: "Enter a valid email address" });
    expect(checkSignUp({ ...GOOD, phone: "0912 34" })).toEqual({ phone: "Check the phone number" });
    expect(checkSignUp({ ...GOOD, password: "abc1234" })).toEqual({ password: "Use at least 8 characters" });
  });

  it("accepts the boundaries the shop accepts", () => {
    expect(checkSignUp(GOOD)).toEqual({});
    expect(checkSignUp({ ...GOOD, name: "a".repeat(100) })).toEqual({});
    expect(checkSignUp({ ...GOOD, name: "a".repeat(101) })).toEqual({ name: "Use at most 100 characters" });
    expect(checkSignUp({ ...GOOD, phone: "+84 912 345 678" })).toEqual({});
    expect(checkSignUp({ ...GOOD, phone: "012345678901234567890" })).toEqual({ phone: "Check the phone number" });
    expect(checkSignUp({ ...GOOD, password: "p".repeat(100) })).toEqual({});
    expect(checkSignUp({ ...GOOD, password: "p".repeat(101) })).toEqual({ password: "Use at most 100 characters" });
    expect(checkSignUp({ ...GOOD, email: `${"a".repeat(188)}@example.com` })).toEqual({});
    expect(checkSignUp({ ...GOOD, email: `${"a".repeat(189)}@example.com` })).toEqual({ email: "Enter a valid email address" });
  });

  it("never trims a password", () => {
    expect(checkSignUp({ ...GOOD, password: "        " })).toEqual({});
  });
});

describe("sign-in rules (SC-22)", () => {
  it("names a blank email and a blank password", () => {
    expect(checkSignIn({ email: "  ", password: "" })).toEqual({
      email: "An email address is needed",
      password: "A password is needed",
    });
    expect(checkSignIn({ email: "lan@example.com", password: "x" })).toEqual({});
  });
});

describe("saved details rules (SC-31)", () => {
  const ok = { name: "Lan Nguyen", phone: "0912 345 678", address: "12 Example Street, Ward 1", city: "Da Nang" };

  it("accepts empty address and city", () => {
    expect(checkDetails({ ...ok, address: "", city: "" })).toEqual({});
  });

  it("names each field at fault under its own sentence", () => {
    expect(checkDetails({ ...ok, phone: "01234567" })).toEqual({ phone: "Check the phone number" });
    expect(checkDetails({ ...ok, name: "" })).toEqual({ name: "Fill in your name" });
    expect(checkDetails({ ...ok, address: "a".repeat(201) })).toEqual({ address: "Use at most 200 characters" });
    expect(checkDetails({ ...ok, city: "c".repeat(101) })).toEqual({ city: "Use at most 100 characters" });
    expect(checkDetails({ ...ok, address: "a".repeat(200), city: "c".repeat(100) })).toEqual({});
  });
});
