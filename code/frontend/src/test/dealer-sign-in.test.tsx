// @vitest-environment jsdom
import { vi } from "vitest";

// The API client's two calls, standing in for /v1/auth/*. Hoisted so the module
// the session imports is replaced before it is first read. The sandbox can run
// with NODE_ENV=production, whose React has no act(): the stub has to land
// before React loads.
const stubs = vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
  return { get: vi.fn(), post: vi.fn(), signIn: vi.fn(), signOut: vi.fn() };
});
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: { get: stubs.get, post: stubs.post } };
});

import { act, cleanup, fireEvent, render, renderHook, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import AdminGate from "../components/admin/AdminGate";
import { SignInForm } from "../components/admin/SignInPage";
import { ApiError } from "../lib/api";
import * as session from "../lib/session";
import { signIn, signOut, useRole } from "../lib/session";

const FORGED = JSON.stringify({ email: "forged@example.com", name: "Forged", role: "admin" });

const account = (role: string) => ({ email: `${role}@example.com`, name: "Sam", role });
const refused = (status: number) =>
  new ApiError(status, status === 429 ? "rate_limited" : status === 401 ? "invalid_credentials" : "internal", "refused");

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

/** Somebody of this role is signed in, as the server's answer says. */
async function signedInAs(role: string) {
  stubs.signIn.mockResolvedValueOnce(account(role));
  await act(async () => {
    await signIn(`${role}@example.com`, "a long enough password");
  });
}

async function nobody() {
  await act(async () => {
    await signOut();
  });
}

function gate() {
  return render(
    <MemoryRouter initialEntries={["/admin/orders"]}>
      <Routes>
        <Route path="/admin/sign-in" element={<p>the sign-in page</p>} />
        <Route path="/" element={<p>the shop</p>} />
        <Route path="/admin/orders" element={<AdminGate><p>the orders</p></AdminGate>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  Object.values(stubs).forEach((fn) => fn.mockReset());
  stubs.post.mockImplementation((path: string, body?: unknown) => (path === "auth/sign-out" ? stubs.signOut() : stubs.signIn(body)));
  stubs.signOut.mockResolvedValue(undefined);
});

afterEach(cleanup);

// The first test is the only one that sees the session before the server has
// answered: the answer is kept for the rest of the file, as it is in the page.
describe("the gate on /admin/*", () => {
  it("draws nothing and goes nowhere while the server is asked, then takes an unreachable one for nobody", async () => {
    // A forged entry in the browser's storage opens nothing.
    localStorage.setItem("agentteam:session", FORGED);
    const asked = deferred<unknown>();
    stubs.get.mockReturnValueOnce(asked.promise);
    const { container } = gate();
    expect(container.innerHTML).toBe("");
    expect(stubs.get).toHaveBeenCalledTimes(1);
    expect(stubs.get).toHaveBeenCalledWith("auth/me");
    await act(async () => {
      await Promise.resolve();
    });
    expect(container.innerHTML).toBe("");
    await act(async () => {
      asked.reject(new ApiError(0, "network", "The server cannot be reached."));
    });
    expect(screen.getByText("the sign-in page")).toBeTruthy();
    expect(screen.queryByText("the orders")).toBeNull();
  });

  it.each(["admin", "staff"])("lets a %s account in", async (role) => {
    await signedInAs(role);
    gate();
    expect(screen.getByText("the orders")).toBeTruthy();
  });

  it("gives a customer the plain message and a way home, never the pages", async () => {
    await signedInAs("customer");
    gate();
    expect(screen.getByText("This area is for the shop's team")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Back to the shop" }).getAttribute("href")).toBe("/");
    expect(screen.queryByText("the orders")).toBeNull();
  });

  it("sends nobody to the sign-in page and tells the server", async () => {
    await signedInAs("staff");
    await nobody();
    gate();
    expect(screen.getByText("the sign-in page")).toBeTruthy();
    expect(screen.queryByText("the orders")).toBeNull();
    expect(stubs.signOut).toHaveBeenCalledTimes(1);
    expect(stubs.post).toHaveBeenLastCalledWith("auth/sign-out");
  });
});

describe("the session module", () => {
  it("still exports its five names", () => {
    for (const name of ["useSession", "signIn", "signOut", "useRole", "allowed"] as const) {
      expect(typeof session[name]).toBe("function");
    }
  });

  it.each(["admin", "staff", "customer"])("reads the role %s from the server's answer", async (role) => {
    await signedInAs(role);
    expect(renderHook(() => useRole()).result.current).toBe(role);
  });

  it("reads no role for nobody", async () => {
    await nobody();
    expect(renderHook(() => useRole()).result.current).toBe("");
  });

  it("answers each refusal with the word the form acts on, and signs nobody in", async () => {
    await nobody();
    const cases: [number, string][] = [[401, "wrong"], [422, "wrong"], [429, "limited"], [500, "unreachable"], [503, "unreachable"], [0, "unreachable"]];
    for (const [status, word] of cases) {
      stubs.signIn.mockRejectedValueOnce(refused(status));
      expect(await signIn("a@example.com", "pw")).toBe(word);
    }
    stubs.signIn.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    expect(await signIn("a@example.com", "pw")).toBe("unreachable");
    stubs.signIn.mockResolvedValueOnce({ not: "an account" });
    expect(await signIn("a@example.com", "pw")).toBe("unreachable");
    expect(renderHook(() => useRole()).result.current).toBe("");
  });

  it("trims the email it sends and leaves the password as typed", async () => {
    await signedInAs("staff");
    stubs.signIn.mockResolvedValueOnce(account("staff"));
    await act(async () => {
      await signIn("  Minh@Example.com ", " pw ");
    });
    expect(stubs.post).toHaveBeenLastCalledWith("auth/sign-in", { email: "Minh@Example.com", password: " pw " });
  });

  it("checks the visitor against the role list, and nobody against any", () => {
    const staff = { email: "s@example.com", name: "S", role: "staff" as const };
    expect(session.allowed(staff, ["Admin", "staff"])).toBe(true);
    expect(session.allowed(staff, ["admin"])).toBe(false);
    expect(session.allowed(staff)).toBe(true);
    expect(session.allowed(null, [])).toBe(false);
  });

  it("never wrote to browser storage", () => {
    expect(localStorage.getItem("agentteam:session")).toBe(FORGED);
    expect(localStorage.length).toBe(1);
    expect(sessionStorage.length).toBe(0);
  });
});

function form() {
  return render(
    <MemoryRouter initialEntries={["/admin/sign-in"]}>
      <Routes>
        <Route path="/admin/sign-in" element={<SignInForm to="/admin" note="the note" />} />
        <Route path="/admin" element={<p>the overview</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

const field = (label: string) => screen.getByLabelText(label) as HTMLInputElement;
const type = (label: string, value: string) => fireEvent.change(field(label), { target: { value } });
const button = () => document.querySelector("form button[type='submit']") as HTMLButtonElement;
const press = () => act(async () => {
  fireEvent.click(button());
});

describe("the sign-in form", () => {
  it("stops a blank email and a blank password, and sends nothing", async () => {
    await nobody();
    form();
    await press();
    expect(screen.getByText("An email address is needed")).toBeTruthy();
    type("Email", "x@example.com");
    await press();
    expect(screen.queryByText("An email address is needed")).toBeNull();
    expect(screen.getByText("A password is needed")).toBeTruthy();
    expect(stubs.signIn).not.toHaveBeenCalled();
    expect(button().textContent).toBe("Sign in");
  });

  it("says one thing for a refusal, empties the password and keeps the email", async () => {
    await nobody();
    stubs.signIn.mockRejectedValueOnce(refused(401));
    form();
    type("Email", "dealer@example.com");
    type("Password", "not-the-password");
    await press();
    expect(screen.getByRole("alert").textContent).toBe("Wrong email or password.");
    expect(field("Password").value).toBe("");
    expect(field("Email").value).toBe("dealer@example.com");
    expect(button().disabled).toBe(false);
  });

  it("says the shop cannot be reached and keeps what was typed", async () => {
    await nobody();
    stubs.signIn.mockRejectedValueOnce(new ApiError(0, "network", "The server cannot be reached."));
    form();
    type("Email", "dealer@example.com");
    type("Password", "Dealer-Pass1");
    await press();
    expect(screen.getByRole("alert").textContent).toBe("We cannot reach the server. Please try again.");
    expect(field("Email").value).toBe("dealer@example.com");
    expect(field("Password").value).toBe("Dealer-Pass1");
    expect(button().disabled).toBe(false);
  });

  it("says to wait after too many tries", async () => {
    await nobody();
    stubs.signIn.mockRejectedValueOnce(refused(429));
    form();
    type("Email", "dealer@example.com");
    type("Password", "Dealer-Pass1");
    await press();
    expect(screen.getByRole("alert").textContent).toBe("Too many tries. Please wait 15 minutes.");
    expect(button().disabled).toBe(false);
  });

  it("disables the button while the shop answers, and sends one request", async () => {
    await nobody();
    const pending = deferred<unknown>();
    stubs.signIn.mockReturnValueOnce(pending.promise);
    form();
    type("Email", "dealer@example.com");
    type("Password", "not-the-password");
    await press();
    expect(button().textContent).toBe("Signing in…");
    expect(button().disabled).toBe(true);
    await act(async () => {
      fireEvent.submit(button().form as HTMLFormElement);
    });
    expect(stubs.signIn).toHaveBeenCalledTimes(1);
    await act(async () => {
      pending.reject(refused(401));
    });
    expect(button().textContent).toBe("Sign in");
    expect(button().disabled).toBe(false);
    expect(screen.getByRole("alert").textContent).toBe("Wrong email or password.");
  });

  it("goes to the area when the shop lets the person in", async () => {
    await nobody();
    stubs.signIn.mockResolvedValueOnce(account("staff"));
    form();
    type("Email", "staff@example.com");
    type("Password", "a long enough password");
    await press();
    expect(screen.getByText("the overview")).toBeTruthy();
  });

  it("offers Continue only to somebody the server says is signed in", async () => {
    await nobody();
    const { unmount } = form();
    expect(screen.queryByText(/Signed in as/)).toBeNull();
    unmount();
    await signedInAs("staff");
    form();
    expect(screen.getByText(/Signed in as staff@example\.com\./)).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    });
    expect(screen.getByText("the overview")).toBeTruthy();
  });
});
