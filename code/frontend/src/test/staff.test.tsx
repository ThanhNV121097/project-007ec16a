import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

vi.mock("../lib/staff", async (original) => ({
  ...(await original<typeof import("../lib/staff")>()),
  useStaff: vi.fn(),
}));
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui

import StaffAccounts from "../components/StaffAccounts";
import StaffDialogs from "../components/StaffDialogs";
import { signIn, signOut } from "../lib/session";
=======
vi.mock("../lib/session", async (original) => ({
  ...(await original<typeof import("../lib/session")>()),
  useRole: vi.fn(),
}));

import StaffAccounts from "../components/StaffAccounts";
import StaffDialogs from "../components/StaffDialogs";
import { useRole } from "../lib/session";
>>>>>>> refs/heads/main
import { useStaff, type StaffAccount, type StaffResult } from "../lib/staff";
import AdminStaff from "../pages/AdminStaff";

afterEach(() => {
  cleanup();
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
  signOut();
=======
>>>>>>> refs/heads/main
  vi.clearAllMocks();
});

const team: StaffAccount[] = [
  { id: "a", name: "Shop owner", email: "owner@example.com", role: "admin", is_active: true },
  { id: "b", name: "Minh Nguyen", email: "minh@example.com", role: "staff", is_active: true },
  { id: "c", name: "Hoa Pham", email: "hoa@example.com", role: "staff", is_active: false },
];

<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
describe("the staff table", () => {
  it("says it is loading and draws no table", () => {
    render(<StaffAccounts items={[]} status="loading" onRetry={() => undefined} onChoose={() => undefined} />);
=======
const none = () => undefined;

describe("the staff table", () => {
  it("says it is loading and draws no table", () => {
    render(<StaffAccounts items={[]} total={0} status="loading" onRetry={none} onChoose={none} />);
>>>>>>> refs/heads/main
    expect(screen.getByText("Loading the staff…")).toBeInTheDocument();
    expect(screen.queryByRole("table")).toBeNull();
  });

  it("says it could not load, draws no table, and asks again on Try again", () => {
    const retry = vi.fn();
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    render(<StaffAccounts items={[]} status="error" onRetry={retry} onChoose={() => undefined} />);
=======
    render(<StaffAccounts items={[]} total={0} status="error" onRetry={retry} onChoose={none} />);
>>>>>>> refs/heads/main
    expect(screen.getByText("We could not load the staff.")).toBeInTheDocument();
    expect(screen.queryByRole("table")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });

  it("draws the admin first with no menu, and a menu on each staff row", () => {
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    render(<StaffAccounts items={team} status="ready" onRetry={() => undefined} onChoose={() => undefined} />);
=======
    render(<StaffAccounts items={team} total={3} status="ready" onRetry={none} onChoose={none} />);
>>>>>>> refs/heads/main
    const rows = screen.getAllByRole("row").slice(1);
    expect(rows).toHaveLength(3);
    expect(rows[0]).toHaveTextContent("Shop owner");
    expect(rows[0]).toHaveTextContent("Admin");
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    expect(rows[0]).toHaveTextContent("Active");
=======
>>>>>>> refs/heads/main
    expect(rows[0].querySelector("button")).toBeNull();
    expect(rows[1]).toHaveTextContent("Staff");
    expect(rows[1].querySelector("button")).not.toBeNull();
    expect(rows[2]).toHaveTextContent("Switched off");
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    expect(screen.getAllByRole("columnheader").map((h) => h.textContent)).toEqual(["Name", "Email", "Role", "Status", "Actions"]);
=======
>>>>>>> refs/heads/main
  });
});

describe("the add dialog", () => {
  const actions = (add: () => Promise<StaffResult>) => ({ add, update: vi.fn(), remove: vi.fn() });
  const open = (add: () => Promise<StaffResult>) =>
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    render(<StaffDialogs dialog={{ kind: "add" }} actions={actions(add)} onClose={() => undefined} onSaved={() => undefined} />);
=======
    render(<StaffDialogs dialog={{ kind: "add" }} actions={actions(add)} onClose={none} onSaved={none} />);
>>>>>>> refs/heads/main
  const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });

  it("has three labelled fields, the password hidden, with its hint", () => {
    open(() => Promise.resolve({ ok: true }));
    expect(screen.getByRole("dialog", { name: "Add staff member" })).toBeInTheDocument();
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Starting password")).toHaveAttribute("type", "password");
    expect(screen.getByText("At least 10 characters.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });

  it("shows each refusal beside its own field, marks it invalid and keeps what was typed", async () => {
    open(() =>
      Promise.resolve({
        ok: false,
        kind: "fields",
        errors: { name: "A name is needed", password: "The password needs at least 10 characters." },
      }),
    );
    type("Email", "lan@example.com");
    type("Starting password", "short");
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    const name = await screen.findByText("A name is needed");
    expect(name).toHaveAttribute("role", "alert");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-describedby", name.id);
    expect(screen.getByText("The password needs at least 10 characters.")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).not.toHaveAttribute("aria-invalid");
    expect(screen.getByLabelText("Email")).toHaveValue("lan@example.com");
    expect(screen.getByLabelText("Starting password")).toHaveValue("short");
  });

  it("reads Saving… with the button disabled while it waits, and sends once", async () => {
    const add = vi.fn(() => new Promise<StaffResult>(() => undefined));
    open(add);
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    const saving = await screen.findByRole("button", { name: "Saving…" });
    expect(saving).toBeDisabled();
=======
    expect(screen.getByLabelText("Starting password")).toHaveAttribute("type", "password");
    expect(screen.getByText("At least 10 characters.")).toBeInTheDocument();
  });

  it("shows each refusal beside its own field and keeps what was typed", async () => {
    open(() => Promise.resolve({ ok: false, kind: "fields", errors: { name: "A name is needed" } }));
    type("Email", "lan@example.com");
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    const name = await screen.findByText("A name is needed");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-describedby", name.id);
    expect(screen.getByLabelText("Email")).toHaveValue("lan@example.com");
  });

  it("reads Saving… while it waits and sends once", async () => {
    const add = vi.fn(() => new Promise<StaffResult>(none));
    open(add);
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(await screen.findByRole("button", { name: "Saving…" })).toBeDisabled();
>>>>>>> refs/heads/main
    fireEvent.submit(screen.getByLabelText("Name").closest("form") as HTMLFormElement);
    expect(add).toHaveBeenCalledTimes(1);
  });

<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
  it("stays open with the values and says it could not save when the shop cannot be reached", async () => {
=======
  it("says it could not save when the shop cannot be reached", async () => {
>>>>>>> refs/heads/main
    open(() => Promise.resolve({ ok: false, kind: "unreachable" }));
    type("Name", "Lan Tran");
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Lan Tran");
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
=======
>>>>>>> refs/heads/main
  });
});

describe("the questions", () => {
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
  const ask = (kind: "off" | "remove", update: () => Promise<StaffResult>, onClose = () => undefined) =>
    render(
      <StaffDialogs
        dialog={{ kind, account: team[1] }}
        actions={{ add: vi.fn(), update, remove: update }}
        onClose={onClose}
        onSaved={() => undefined}
      />,
    );

  it("names the person in the switch-off question", () => {
    ask("off", () => Promise.resolve({ ok: true }));
    expect(screen.getByRole("alertdialog", { name: "Switch off Minh Nguyen?" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Switch off" })).toBeInTheDocument();
  });

  it("names the person in the removal question", () => {
    ask("remove", () => Promise.resolve({ ok: true }));
    expect(screen.getByRole("alertdialog", { name: "Remove Minh Nguyen?" })).toBeInTheDocument();
    expect(screen.getByText("The account is deleted and cannot be restored. Their work stays.")).toBeInTheDocument();
=======
  const ask = (kind: "off" | "remove", update: () => Promise<StaffResult>, onClose = none) =>
    render(<StaffDialogs dialog={{ kind, account: team[1] }} actions={{ add: vi.fn(), update, remove: update }} onClose={onClose} onSaved={none} />);

  it("names the person in each question", () => {
    ask("off", () => Promise.resolve({ ok: true }));
    expect(screen.getByRole("alertdialog", { name: "Switch off Minh Nguyen?" })).toBeInTheDocument();
    cleanup();
    ask("remove", () => Promise.resolve({ ok: true }));
    expect(screen.getByRole("alertdialog", { name: "Remove Minh Nguyen?" })).toBeInTheDocument();
>>>>>>> refs/heads/main
  });

  it("stays open and shows the shop's sentence when it refuses", async () => {
    const onClose = vi.fn();
    ask("off", () => Promise.resolve({ ok: false, kind: "refused", message: "That account was not found." }), onClose);
    fireEvent.click(screen.getByRole("button", { name: "Switch off" }));
    expect(await screen.findByText("That account was not found.")).toHaveAttribute("role", "alert");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes without asking the shop anything on Cancel", () => {
    const update = vi.fn();
    const onClose = vi.fn();
    ask("remove", update, onClose);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(update).not.toHaveBeenCalled();
  });
});

describe("who the page is for", () => {
  beforeEach(() => {
    vi.mocked(useStaff).mockReturnValue({
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
      items: [], status: "loading", retry: vi.fn(), add: vi.fn(), update: vi.fn(), remove: vi.fn(),
=======
      items: [], total: 0, status: "loading", retry: vi.fn(), add: vi.fn(), update: vi.fn(), remove: vi.fn(),
>>>>>>> refs/heads/main
    });
  });

  it("sends a staff session to /admin without asking for the list", () => {
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    signIn("minh@example.com", "Minh Nguyen", "staff");
=======
    vi.mocked(useRole).mockReturnValue("staff");
>>>>>>> refs/heads/main
    render(
      <MemoryRouter initialEntries={["/admin/staff"]}>
        <Routes>
          <Route path="/admin/staff" element={<AdminStaff />} />
          <Route path="/admin" element={<p>overview reached</p>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("overview reached")).toBeInTheDocument();
    expect(useStaff).not.toHaveBeenCalled();
<<<<<<< refs/heads/feature/staff-accounts-managed-by-the-owner-ui
    expect(screen.queryByText("Staff")).toBeNull();
=======
>>>>>>> refs/heads/main
  });
});
