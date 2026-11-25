import { act } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

// The sandbox that runs the suite sets NODE_ENV=production, and React's act()
// exists only in the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

import AccountEntry from "../components/account/AccountEntry";
import AccountView from "../components/account/AccountView";
import SignInView from "../components/account/SignInView";
import SignUpView from "../components/account/SignUpView";
import { customerSignIn, customerSignOut } from "../lib/customer-accounts-sign-up-sign-in-my-orders";

afterEach(cleanup);

const signedInAs = (email: string) => act(async () => { await customerSignIn(email, "anything"); });
const signedOut = () => act(async () => { await customerSignOut(); });
const type = (el: HTMLElement, value: string) => fireEvent.change(el, { target: { value } });

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

describe("the header's account entry (SC-35, SC-36)", () => {
  it("draws nothing while /v1/auth/me is asked, then Sign in, then the customer's name", async () => {
    render(<MemoryRouter><AccountEntry variant="bar" /></MemoryRouter>);
    expect(screen.queryByTestId("account-entry")).toBeNull();
    const entry = await screen.findByTestId("account-entry");
    expect(entry).toHaveTextContent("Sign in");
    expect(entry).toHaveAttribute("href", "/account/sign-in");
    await signedInAs("lan@example.com");
    await waitFor(() => expect(screen.getByTestId("account-entry")).toHaveTextContent("Lan Nguyen"));
    expect(screen.getByTestId("account-entry")).toHaveAttribute("href", "/account");
    await signedInAs("minh@example.com");
    await waitFor(() => expect(screen.getByTestId("account-entry")).toHaveAttribute("href", "/admin"));
  });
});

describe("sign up (SC-18, SC-19)", () => {
  it("labels four fields and names every empty one when the button is pressed", () => {
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
  });

  it("refuses an email that already belongs to an account and empties only the password", async () => {
    render(<MemoryRouter><SignUpView /></MemoryRouter>);
    type(screen.getByLabelText("Name"), "Other Lan");
    type(screen.getByLabelText("Email"), "LAN@example.com");
    type(screen.getByLabelText("Phone"), "0912 345 678");
    type(screen.getByLabelText("Password"), "lanpass123");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("An account with this email already exists. Sign in instead.")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveValue("LAN@example.com");
    expect(screen.getByLabelText("Phone")).toHaveValue("0912 345 678");
    expect(screen.getByLabelText("Password")).toHaveValue("");
    expect(screen.getByRole("button", { name: "Create account" })).toBeEnabled();
  });

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
  it("names a blank email and a blank password and sends nothing", () => {
    render(<MemoryRouter><SignInView /></MemoryRouter>);
    expect(screen.getByRole("heading", { level: 1, name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByText("Buying as a guest needs no account.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Create an account" })).toHaveAttribute("href", "/account/sign-up");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(screen.getAllByRole("alert").map((a) => a.textContent)).toEqual(["An email address is needed", "A password is needed"]);
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
});

describe("My account (SC-25, SC-30, SC-32, SC-34)", () => {
  it("lists the orders, saves details, adds a guest order and signs out", async () => {
    await signedInAs("lan@example.com");
    account();
    expect(await screen.findByRole("heading", { level: 1, name: "My account" })).toBeInTheDocument();
    expect(screen.getByText("Hello, Lan Nguyen")).toBeInTheDocument();
    expect(screen.getByText("Loading your orders…")).toBeInTheDocument();
    const rows = await screen.findAllByTestId("order-row");
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveAttribute("href", "/orders/doi-24817");
    expect(rows[0]).toHaveTextContent("DOI-24817");
    expect(rows[0]).toHaveTextContent(/\d{2}\/\d{2}\/\d{4}/);
    expect(rows[0]).toHaveTextContent(/6\.180\.000/);
    expect(rows[0]).toHaveTextContent("On its way");

    await waitFor(() => expect(document.getElementById("details-phone")).toHaveValue("0912 345 678"));
    type(screen.getByLabelText("Street address, ward, district"), "12 Example Street, Ward 1");
    type(screen.getByLabelText("City or province"), "Da Nang");
    fireEvent.click(screen.getByRole("button", { name: "Save details" }));
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();

    type(screen.getByLabelText("Order number"), "doi 24819");
    type(document.getElementById("add-order-phone") as HTMLElement, "+84912345678");
    fireEvent.click(screen.getByRole("button", { name: "Add order" }));
    const refused = await screen.findByText(/No order matches that number and phone/);
    expect(refused.closest("[role=alert]")).toHaveTextContent("No order matches that number and phone. Check both, or message us.");
    expect(screen.getByRole("link", { name: "message us" })).toHaveAttribute("href", "/contact");

    type(screen.getByLabelText("Order number"), "doi 24818");
    fireEvent.click(screen.getByRole("button", { name: "Add order" }));
    expect(await screen.findByText("Added.")).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByTestId("order-row")).toHaveLength(3));

    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(await screen.findByText("home screen")).toBeInTheDocument();
  });

  it("goes to sign-in for a visitor and to /admin for the shop's team", async () => {
    await signedOut();
    account();
    expect(await screen.findByText("sign-in screen")).toBeInTheDocument();
    cleanup();
    await signedInAs("minh@example.com");
    account();
    expect(await screen.findByText("team area")).toBeInTheDocument();
  });
});
