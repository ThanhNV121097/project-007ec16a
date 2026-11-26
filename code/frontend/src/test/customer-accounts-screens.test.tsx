<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
// @vitest-environment jsdom
import { vi } from "vitest";

// The API client's calls, standing in for the shop. Hoisted so the module the
// session and the story's client import is replaced before it is first read.
// The sandbox can run with NODE_ENV=production, whose React has no act(): the
// stub has to land before React loads.
const shop = vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
  return { get: vi.fn(), post: vi.fn(), put: vi.fn() };
});
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: { get: shop.get, post: shop.post, put: shop.put } };
});

import { act } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
=======
import { act } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

// The sandbox that runs the suite sets NODE_ENV=production, and React's act()
// exists only in the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

>>>>>>> refs/heads/main
import AccountEntry from "../components/account/AccountEntry";
import AccountView from "../components/account/AccountView";
import SignInView from "../components/account/SignInView";
import SignUpView from "../components/account/SignUpView";
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
import { ApiError } from "../lib/api";
import { refreshSession } from "../lib/session";

const LAN = { email: "lan@example.com", name: "Lan Nguyen", role: "customer" };
const MINH = { email: "minh@example.com", name: "Minh", role: "staff" };
const row = (n: number) => ({ id: `doi-${n}`, number: `DOI-${n}`, placed_at: "2026-11-25T10:00:00+07:00", status: "shipped", total: 6180000 });
const saved = { name: "Lan Nguyen", email: "lan@example.com", phone: "0912 345 678", address: "", city: "" };
const refused = (status: number, code: string, errors?: Record<string, string>) => new ApiError(status, code, "refused", errors);
const unreachable = "We cannot reach the server. Please try again.";

let who: typeof LAN | typeof MINH | null = null;
let rows = [row(24817), row(24816)];

beforeEach(() => {
  who = null;
  rows = [row(24817), row(24816)];
  Object.values(shop).forEach((fn) => fn.mockReset());
  shop.get.mockImplementation(async (path: string) => {
    if (path === "auth/me") {
      if (who) return who;
      throw refused(401, "unauthenticated");
    }
    if (path === "account") return saved;
    if (path === "account/orders") return { items: rows, total: rows.length, page: 1, limit: 10 };
    throw new Error(`unexpected GET ${path}`);
  });
  shop.post.mockImplementation(async (path: string) => {
    if (path === "auth/sign-out") {
      who = null;
      return undefined;
    }
    throw new Error(`unexpected POST ${path}`);
  });
});

afterEach(cleanup);

/** The shop now says this person is signed in (or nobody): the one cache is asked again. */
const thenAsk = (person: typeof who) => act(async () => {
  who = person;
  await refreshSession();
});
const type = (el: HTMLElement, value: string) => fireEvent.change(el, { target: { value } });
/** A call that stays open until the test settles it. */
function held() {
  let reject!: (e: unknown) => void;
  let resolve!: (v: unknown) => void;
  const promise = new Promise((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}
=======
import { customerSignIn, customerSignOut } from "../lib/customer-accounts-sign-up-sign-in-my-orders";

afterEach(cleanup);

const signedInAs = (email: string) => act(async () => { await customerSignIn(email, "anything"); });
const signedOut = () => act(async () => { await customerSignOut(); });
const type = (el: HTMLElement, value: string) => fireEvent.change(el, { target: { value } });
>>>>>>> refs/heads/main

function account() {
  return render(
    <MemoryRouter initialEntries={["/account"]}>
      <Routes>
        <Route path="/account" element={<AccountView />} />
        <Route path="/account/sign-in" element={<p>sign-in screen</p>} />
        <Route path="/admin" element={<p>team area</p>} />
        <Route path="/" element={<p>home screen</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
// The first test is the only one that sees the session before the shop has
// answered: the answer is kept for the rest of the file, as it is in the page.
describe("the header's account entry (SC-35, SC-36)", () => {
  it("draws nothing while /v1/auth/me is asked, then Sign in, then the customer's name", async () => {
    const asked = held();
    shop.get.mockImplementationOnce(() => asked.promise);
=======
describe("the header's account entry (SC-35, SC-36)", () => {
  it("draws nothing while /v1/auth/me is asked, then Sign in, then the customer's name", async () => {
>>>>>>> refs/heads/main
    render(<MemoryRouter><AccountEntry variant="bar" /></MemoryRouter>);
    expect(screen.queryByTestId("account-entry")).toBeNull();
    // The bar keeps the width of "Sign in" while the shop is asked, but offers no link and hides it from everyone.
    const pending = screen.getByTestId("account-entry-pending");
    expect(pending).toHaveAttribute("aria-hidden", "true");
    expect(pending).toHaveClass("invisible");
    expect(screen.queryByRole("link")).toBeNull();
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    await act(async () => {
      asked.reject(refused(401, "unauthenticated"));
    });
=======
>>>>>>> refs/heads/main
    const entry = await screen.findByTestId("account-entry");
    expect(screen.queryByTestId("account-entry-pending")).toBeNull();
    expect(entry).toHaveTextContent("Sign in");
    expect(entry).toHaveAttribute("href", "/account/sign-in");
    // `truncate` clips to the padding box, so the bar link keeps room under the line for its underline.
    expect(entry).toHaveClass("truncate", "pb-3", "-mb-3", "underline-offset-8");
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    await thenAsk(LAN);
    await waitFor(() => expect(screen.getByTestId("account-entry")).toHaveTextContent("Lan Nguyen"));
    expect(screen.getByTestId("account-entry")).toHaveAttribute("href", "/account");
    await thenAsk(MINH);
=======
    await signedInAs("lan@example.com");
    await waitFor(() => expect(screen.getByTestId("account-entry")).toHaveTextContent("Lan Nguyen"));
    expect(screen.getByTestId("account-entry")).toHaveAttribute("href", "/account");
    await signedInAs("minh@example.com");
>>>>>>> refs/heads/main
    await waitFor(() => expect(screen.getByTestId("account-entry")).toHaveAttribute("href", "/admin"));
  });
});

describe("sign up (SC-18, SC-19)", () => {
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
  const fill = () => {
    type(screen.getByLabelText("Name"), "Other Lan");
    type(screen.getByLabelText("Email"), "LAN@example.com");
    type(screen.getByLabelText("Phone"), "0912 345 678");
    type(screen.getByLabelText("Password"), "lanpass123");
  };

  it("labels four fields and names every empty one when the button is pressed, sending nothing", () => {
=======
  it("labels four fields and names every empty one when the button is pressed", () => {
>>>>>>> refs/heads/main
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    expect(screen.getByRole("heading", { level: 1, name: "Create your account" })).toBeInTheDocument();
    expect(screen.getByText("Optional — you can always buy without one.")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
    expect(screen.getByLabelText("Phone")).toHaveAttribute("inputmode", "tel");
    expect(screen.getByLabelText("Name")).toHaveAttribute("autocomplete", "name");
    expect(screen.getByLabelText("Email")).toHaveAttribute("autocomplete", "email");
    expect(screen.getByLabelText("Password")).toHaveAttribute("autocomplete", "new-password");
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute("href", "/account/sign-in");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(screen.getAllByRole("alert").map((a) => a.textContent)).toEqual([
      "Fill in your name",
      "Enter a valid email address",
      "Check the phone number",
      "Use at least 8 characters",
    ]);
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-invalid", "true");
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    expect(shop.post).not.toHaveBeenCalled();
  });

  it("sends the trimmed values, refuses an email that belongs to an account and empties only the password", async () => {
    shop.post.mockRejectedValueOnce(refused(409, "conflict", { email: "An account with this email already exists. Sign in instead." }));
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    fill();
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("An account with this email already exists. Sign in instead.")).toBeInTheDocument();
    expect(shop.post).toHaveBeenCalledWith("account/sign-up", { name: "Other Lan", email: "LAN@example.com", phone: "0912 345 678", password: "lanpass123" });
=======
  });

  it("refuses an email that already belongs to an account and empties only the password", async () => {
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    type(screen.getByLabelText("Name"), "Other Lan");
    type(screen.getByLabelText("Email"), "LAN@example.com");
    type(screen.getByLabelText("Phone"), "0912 345 678");
    type(screen.getByLabelText("Password"), "lanpass123");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("An account with this email already exists. Sign in instead.")).toBeInTheDocument();
>>>>>>> refs/heads/main
    expect(screen.getByLabelText("Email")).toHaveValue("LAN@example.com");
    expect(screen.getByLabelText("Phone")).toHaveValue("0912 345 678");
    expect(screen.getByLabelText("Password")).toHaveValue("");
    expect(screen.getByRole("button", { name: "Create account" })).toBeEnabled();
  });

<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
  it("shows the busy button while the shop answers, then says it cannot be reached and keeps the typed values", async () => {
    const slow = held();
    shop.post.mockImplementationOnce(() => slow.promise);
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    fill();
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(screen.getByRole("button", { name: "Creating your account…" })).toBeDisabled();
    await act(async () => {
      slow.reject(new ApiError(0, "network", "The server cannot be reached."));
    });
    expect(await screen.findByText(unreachable)).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Other Lan");
    expect(screen.getByLabelText("Password")).toHaveValue("");
  });

  it("says to wait after too many tries", async () => {
    shop.post.mockRejectedValueOnce(refused(429, "rate_limited"));
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    fill();
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("Too many tries. Please wait and try again.")).toBeInTheDocument();
  });
});

describe("sign in (SC-22, SC-24)", () => {
  const attempt = () => {
    type(screen.getByLabelText("Email"), "lan@example.com");
    type(screen.getByLabelText("Password"), "wrong-password");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
  };

=======
  it("says the shop cannot be reached and keeps the typed values", async () => {
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    type(screen.getByLabelText("Name"), "Lan");
    type(screen.getByLabelText("Email"), "down@example.com");
    type(screen.getByLabelText("Phone"), "0912 345 678");
    type(screen.getByLabelText("Password"), "lanpass123");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(screen.getByRole("button", { name: "Creating your account…" })).toBeDisabled();
    expect(await screen.findByText("We cannot reach the server. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Lan");
    expect(screen.getByLabelText("Password")).toHaveValue("");
  });
});

describe("sign in (SC-22, SC-24)", () => {
>>>>>>> refs/heads/main
  it("names a blank email and a blank password and sends nothing", () => {
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    expect(screen.getByRole("heading", { level: 1, name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByText("Buying as a guest needs no account.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Create an account" })).toHaveAttribute("href", "/account/sign-up");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(screen.getAllByRole("alert").map((a) => a.textContent)).toEqual(["An email address is needed", "A password is needed"]);
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    expect(shop.post).not.toHaveBeenCalled();
  });

  it("answers a wrong password with one sentence, keeping the email and emptying the password", async () => {
    shop.post.mockRejectedValueOnce(refused(401, "invalid_credentials"));
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    attempt();
    expect(await screen.findByText("Wrong email or password.")).toBeInTheDocument();
    expect(shop.post).toHaveBeenCalledWith("auth/sign-in", { email: "lan@example.com", password: "wrong-password" });
    expect(screen.getByLabelText("Email")).toHaveValue("lan@example.com");
    expect(screen.getByLabelText("Password")).toHaveValue("");
  });

  it("never reads a broken proxy's 401 as a wrong password", async () => {
    shop.post.mockRejectedValueOnce(refused(401, "internal"));
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    attempt();
    expect(await screen.findByText(unreachable)).toBeInTheDocument();
    expect(screen.queryByText("Wrong email or password.")).toBeNull();
  });

  it("says to wait 15 minutes after too many tries", async () => {
    shop.post.mockRejectedValueOnce(refused(429, "rate_limited"));
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    attempt();
    expect(await screen.findByText("Too many tries. Please wait 15 minutes.")).toBeInTheDocument();
  });

  it("asks who is signed in again once the shop lets the person in", async () => {
    shop.post.mockImplementationOnce(async () => {
      who = LAN;
      return LAN;
    });
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    attempt();
    await waitFor(() => expect(shop.get.mock.calls.filter(([p]) => p === "auth/me").length).toBeGreaterThan(0));
    await waitFor(() => expect(screen.getByRole("button", { name: "Sign in" })).toBeEnabled());
  });
=======
  });

  it("answers a wrong password with one sentence, keeping the email and emptying the password", async () => {
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    type(screen.getByLabelText("Email"), "lan@example.com");
    type(screen.getByLabelText("Password"), "wrong-password");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByText("Wrong email or password.")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveValue("lan@example.com");
    expect(screen.getByLabelText("Password")).toHaveValue("");
  });
>>>>>>> refs/heads/main
});

describe("My account (SC-25, SC-30, SC-32, SC-34)", () => {
  it("lists the orders, saves details, adds a guest order and signs out", async () => {
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    await thenAsk(LAN);
    const loading = held();
    // The first read of the orders stays open; which of the two reads (the saved details, the orders) goes first does not matter.
    let firstRead = true;
    shop.get.mockImplementation(async (path: string) => {
      if (path === "auth/me") return who;
      if (path === "account") return saved;
      if (path === "account/orders") {
        if (!firstRead) return { items: rows, total: rows.length, page: 1, limit: 10 };
        firstRead = false;
        return loading.promise;
      }
      throw new Error(`unexpected GET ${path}`);
    });
=======
    await signedInAs("lan@example.com");
>>>>>>> refs/heads/main
    account();
    expect(await screen.findByRole("heading", { level: 1, name: "My account" })).toBeInTheDocument();
    expect(screen.getByText("Hello, Lan Nguyen")).toBeInTheDocument();
    expect(screen.getByText("Loading your orders…")).toBeInTheDocument();
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    await act(async () => {
      loading.resolve({ items: rows, total: rows.length, page: 1, limit: 10 });
    });
    const list = await screen.findAllByTestId("order-row");
    expect(list).toHaveLength(2);
    expect(list[0]).toHaveAttribute("href", "/orders/doi-24817");
    expect(list[0]).toHaveTextContent("DOI-24817");
    expect(list[0]).toHaveTextContent(/\d{2}\/\d{2}\/\d{4}/);
    expect(list[0]).toHaveTextContent(/6\.180\.000/);
    expect(list[0]).toHaveTextContent("On its way");

    await waitFor(() => expect(document.getElementById("details-phone")).toHaveValue("0912 345 678"));
    const saving = held();
    shop.put.mockImplementationOnce(() => saving.promise);
=======
    const rows = await screen.findAllByTestId("order-row");
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveAttribute("href", "/orders/doi-24817");
    expect(rows[0]).toHaveTextContent("DOI-24817");
    expect(rows[0]).toHaveTextContent(/\d{2}\/\d{2}\/\d{4}/);
    expect(rows[0]).toHaveTextContent(/6\.180\.000/);
    expect(rows[0]).toHaveTextContent("On its way");

    await waitFor(() => expect(document.getElementById("details-phone")).toHaveValue("0912 345 678"));
>>>>>>> refs/heads/main
    type(screen.getByLabelText("Street address, ward, district"), "12 Example Street, Ward 1");
    type(screen.getByLabelText("City or province"), "Da Nang");
    fireEvent.click(screen.getByRole("button", { name: "Save details" }));
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled();
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    await act(async () => {
      saving.resolve({ ...saved, address: "12 Example Street, Ward 1", city: "Da Nang" });
    });
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(shop.put).toHaveBeenCalledWith("account", { name: "Lan Nguyen", phone: "0912 345 678", address: "12 Example Street, Ward 1", city: "Da Nang" });

    shop.post.mockRejectedValueOnce(refused(404, "not_found"));
    type(screen.getByLabelText("Order number"), "doi 24819");
    type(document.getElementById("add-order-phone") as HTMLElement, "+84912345678");
    fireEvent.click(screen.getByRole("button", { name: "Add order" }));
    const missed = await screen.findByText(/No order matches that number and phone/);
    expect(missed.closest("[role=alert]")).toHaveTextContent("No order matches that number and phone. Check both, or message us.");
    expect(screen.getByRole("link", { name: "message us" })).toHaveAttribute("href", "/contact");
    expect(shop.post).toHaveBeenCalledWith("account/orders", { number: "doi 24819", phone: "+84912345678" });

    shop.post.mockImplementationOnce(async () => {
      rows = [row(24818), ...rows];
      return row(24818);
    });
    type(screen.getByLabelText("Order number"), "doi 24818");
    type(document.getElementById("add-order-phone") as HTMLElement, "+84912345678");
=======
    expect(await screen.findByText("Saved.")).toBeInTheDocument();

    type(screen.getByLabelText("Order number"), "doi 24819");
    type(document.getElementById("add-order-phone") as HTMLElement, "+84912345678");
    fireEvent.click(screen.getByRole("button", { name: "Add order" }));
    const refused = await screen.findByText(/No order matches that number and phone/);
    expect(refused.closest("[role=alert]")).toHaveTextContent("No order matches that number and phone. Check both, or message us.");
    expect(screen.getByRole("link", { name: "message us" })).toHaveAttribute("href", "/contact");

    type(screen.getByLabelText("Order number"), "doi 24818");
>>>>>>> refs/heads/main
    fireEvent.click(screen.getByRole("button", { name: "Add order" }));
    expect(await screen.findByText("Added.")).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByTestId("order-row")).toHaveLength(3));

    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(await screen.findByText("home screen")).toBeInTheDocument();
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
    expect(shop.post).toHaveBeenCalledWith("auth/sign-out");
  });

  it("says the orders could not be loaded and asks again on Try again", async () => {
    await thenAsk(LAN);
    shop.get.mockImplementationOnce(() => Promise.reject(new ApiError(0, "network", "The server cannot be reached.")));
    shop.get.mockImplementationOnce(() => Promise.reject(new ApiError(0, "network", "The server cannot be reached.")));
    account();
    expect(await screen.findByText("We could not load your orders.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect((await screen.findAllByTestId("order-row")).length).toBe(2);
  });

  it("goes to sign-in for a visitor and to /admin for the shop's team", async () => {
    await thenAsk(null);
    account();
    expect(await screen.findByText("sign-in screen")).toBeInTheDocument();
    cleanup();
    await thenAsk(MINH);
=======
  });

  it("goes to sign-in for a visitor and to /admin for the shop's team", async () => {
    await signedOut();
    account();
    expect(await screen.findByText("sign-in screen")).toBeInTheDocument();
    cleanup();
    await signedInAs("minh@example.com");
>>>>>>> refs/heads/main
    account();
    expect(await screen.findByText("team area")).toBeInTheDocument();
  });
});
