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
vi.mock("../lib/session", async (original) => ({
  ...(await original<typeof import("../lib/session")>()),
  useRole: vi.fn(),
}));

import StaffAccounts from "../components/StaffAccounts";
import StaffDialogs from "../components/StaffDialogs";
import { useRole } from "../lib/session";
import { useStaff, type StaffAccount, type StaffResult } from "../lib/staff";
import AdminStaff from "../pages/AdminStaff";

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const team: StaffAccount[] = [
  { id: "a", name: "Shop owner", email: "owner@example.com", role: "admin", is_active: true },
  { id: "b", name: "Minh Nguyen", email: "minh@example.com", role: "staff", is_active: true },
  { id: "c", name: "Hoa Pham", email: "hoa@example.com", role: "staff", is_active: false },
];

const none = () => undefined;

describe("the staff table", () => {
  it("says it is loading and draws no table", () => {
    render(<StaffAccounts items={[]} total={0} status="loading" onRetry={none} onChoose={none} />);
    expect(screen.getByText("Loading the staff…")).toBeInTheDocument();
    expect(screen.queryByRole("table")).toBeNull();
  });

  it("says it could not load, draws no table, and asks again on Try again", () => {
    const retry = vi.fn();
    render(<StaffAccounts items={[]} total={0} status="error" onRetry={retry} onChoose={none} />);
    expect(screen.getByText("We could not load the staff.")).toBeInTheDocument();
    expect(screen.queryByRole("table")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });

  it("draws the admin first with no menu, and a menu on each staff row", () => {
    render(<StaffAccounts items={team} total={3} status="ready" onRetry={none} onChoose={none} />);
    const rows = screen.getAllByRole("row").slice(1);
    expect(rows).toHaveLength(3);
    expect(rows[0]).toHaveTextContent("Shop owner");
    expect(rows[0]).toHaveTextContent("Admin");
    expect(rows[0]).toHaveTextContent("Active");
    expect(rows[0].querySelector("button")).toBeNull();
    expect(rows[1]).toHaveTextContent("Staff");
    expect(rows[1].querySelector("button")).not.toBeNull();
    expect(rows[2]).toHaveTextContent("Switched off");
  });

  it("names the columns Name, Email, Role and Status", () => {
    render(<StaffAccounts items={team} total={3} status="ready" onRetry={none} onChoose={none} />);
    const heads = screen.getAllByRole("columnheader").map((h) => h.textContent);
    expect(heads.slice(0, 4)).toEqual(["Name", "Email", "Role", "Status"]);
  });
});

describe("the add dialog", () => {
  const actions = (add: () => Promise<StaffResult>) => ({ add, update: vi.fn(), remove: vi.fn() });
  const open = (add: () => Promise<StaffResult>) =>
    render(<StaffDialogs dialog={{ kind: "add" }} actions={actions(add)} onClose={none} onSaved={none} />);
  const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });

  it("has three labelled fields, the password hidden, with its hint", () => {
    open(() => Promise.resolve({ ok: true }));
    expect(screen.getByRole("dialog", { name: "Add staff member" })).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Starting password")).toHaveAttribute("type", "password");
    expect(screen.getByText("At least 10 characters.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });

  it("shows each refusal beside its own field and keeps what was typed", async () => {
    open(() => Promise.resolve({ ok: false, kind: "fields", errors: { name: "A name is needed" } }));
    type("Email", "lan@example.com");
    type("Starting password", "short");
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    const name = await screen.findByText("A name is needed");
    expect(name).toHaveAttribute("role", "alert");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-describedby", name.id);
    expect(screen.getByLabelText("Email")).not.toHaveAttribute("aria-invalid");
    expect(screen.getByLabelText("Email")).toHaveValue("lan@example.com");
    expect(screen.getByLabelText("Starting password")).toHaveValue("short");
  });

  it("reads Saving… while it waits and sends once", async () => {
    const add = vi.fn(() => new Promise<StaffResult>(none));
    open(add);
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(await screen.findByRole("button", { name: "Saving…" })).toBeDisabled();
    fireEvent.submit(screen.getByLabelText("Name").closest("form") as HTMLFormElement);
    expect(add).toHaveBeenCalledTimes(1);
  });

  it("says it could not save when the shop cannot be reached, and lets the visitor press Add again", async () => {
    open(() => Promise.resolve({ ok: false, kind: "unreachable" }));
    type("Name", "Lan Tran");
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Lan Tran");
    expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
  });
});

describe("the questions", () => {
  const ask = (kind: "off" | "remove", update: () => Promise<StaffResult>, onClose = none) =>
    render(<StaffDialogs dialog={{ kind, account: team[1] }} actions={{ add: vi.fn(), update, remove: update }} onClose={onClose} onSaved={none} />);

  it("names the person in each question", () => {
    ask("off", () => Promise.resolve({ ok: true }));
    expect(screen.getByRole("alertdialog", { name: "Switch off Minh Nguyen?" })).toBeInTheDocument();
    cleanup();
    ask("remove", () => Promise.resolve({ ok: true }));
    expect(screen.getByRole("alertdialog", { name: "Remove Minh Nguyen?" })).toBeInTheDocument();
    expect(screen.getByText("The account is deleted and cannot be restored. Their work stays.")).toBeInTheDocument();
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
      items: [], total: 0, status: "loading", retry: vi.fn(), add: vi.fn(), update: vi.fn(), remove: vi.fn(),
    });
  });

  it("sends a staff session to /admin without asking for the list", () => {
    vi.mocked(useRole).mockReturnValue("staff");
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
    expect(screen.queryByText("Staff")).toBeNull();
  });
});
