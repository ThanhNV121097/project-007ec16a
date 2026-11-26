import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

vi.mock("../lib/mock/manage-shoes-lines-categories-reviews-and-orders", () => ({
  default: { create: vi.fn(), update: vi.fn(), remove: vi.fn(), read: vi.fn(), upload: vi.fn(), setPairs: vi.fn() },
}));

import DeleteQuestion from "../components/admin/DeleteQuestion";
import RecordDialog from "../components/admin/RecordDialog";
import { fieldsOf } from "../components/admin/fields";
import { deleteRow } from "../components/admin/store";
import transport from "../lib/mock/manage-shoes-lines-categories-reviews-and-orders";
import type { Wire } from "../lib/manage-shoes-lines-categories-reviews-and-orders";

const shop = vi.mocked(transport);
const reply = (status: number, body?: unknown): Wire => ({ status, body });
const none = () => undefined;

/** An answer the test releases when it chooses, to look at the screen while the shop is thinking. */
function held<T>() {
  let release!: (value: T) => void;
  const promise = new Promise<T>((done) => { release = done; });
  return { promise, release };
}

beforeEach(() => {
  Object.values(shop).forEach((fn) => fn.mockReset());
});
afterEach(cleanup);

const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });
const press = (name: string) => fireEvent.click(screen.getByRole("button", { name }));

function openAdd(collection: "lines" | "categories", heading: string, onSaved = none) {
  render(
    <MemoryRouter>
      <RecordDialog collection={collection} specs={fieldsOf(collection)} record={{ id: "" }} heading={heading}
        onClose={none} onSaved={onSaved} />
    </MemoryRouter>,
  );
}

describe("saving from the add dialog", () => {
  it("says Saving…, disables the button, and sends one request however often it is pressed", async () => {
    const answer = held<Wire>();
    shop.create.mockReturnValueOnce(answer.promise);
    const saved = vi.fn();
    openAdd("lines", "Lines", saved);
    expect(screen.getByRole("heading", { name: "Add to lines" })).toBeInTheDocument();
    type("Name", "Samba");
    press("Add");
    const busy = await screen.findByRole("button", { name: "Saving…" });
    expect(busy).toBeDisabled();
    fireEvent.click(busy);
    fireEvent.click(busy);
    expect(shop.create).toHaveBeenCalledTimes(1);
    expect(shop.create).toHaveBeenCalledWith("lines", { name: "Samba" });
    answer.release(reply(201, { id: "samba", name: "Samba" }));
    await waitFor(() => expect(saved).toHaveBeenCalledTimes(1));
  });

  it("asks nothing of the shop while a required field is empty", () => {
    openAdd("lines", "Lines");
    press("Add");
    expect(screen.getByText("Name is needed")).toBeInTheDocument();
    expect(shop.create).not.toHaveBeenCalled();
  });

  it("keeps the dialog and every typed value, with the shop's sentence beside its field", async () => {
    shop.create.mockResolvedValueOnce(reply(422, { error: "invalid", message: "x", errors: { since: "Enter a whole number" } }));
    const saved = vi.fn();
    openAdd("lines", "Lines", saved);
    type("Name", "Samba");
    type("First made", "1949");
    press("Add");
    const sentence = await screen.findByText("Enter a whole number");
    expect(sentence).toHaveAttribute("role", "alert");
    expect(screen.getByLabelText("First made")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Name")).toHaveValue("Samba");
    expect(screen.getByLabelText("First made")).toHaveValue(1949);
    expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
    expect(saved).not.toHaveBeenCalled();
    // Changing the field takes its out-of-date sentence away.
    type("First made", "1950");
    expect(screen.queryByText("Enter a whole number")).toBeNull();
  });

  it("says it could not save when the shop cannot be reached, and lets the person try again", async () => {
    shop.create.mockResolvedValueOnce(reply(0));
    openAdd("lines", "Lines");
    type("Name", "Samba");
    press("Add");
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Samba");
    expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
  });

  it("says the session ended, with a link to sign in, and keeps the values", async () => {
    shop.create.mockResolvedValueOnce(reply(401, { error: "unauthenticated", message: "Sign in." }));
    openAdd("lines", "Lines");
    type("Name", "Samba");
    press("Add");
    const link = await screen.findByRole("link", { name: "Sign in again." });
    expect(link).toHaveAttribute("href", "/admin/sign-in");
    expect(screen.getByText(/Your session ended\./)).toBeInTheDocument();
    expect(screen.queryByText("We could not save. Please try again.")).toBeNull();
    expect(screen.getByLabelText("Name")).toHaveValue("Samba");
  });
});

describe("the picture control", () => {
  const png = () => new File(["x"], "side.png", { type: "image/png" });
  const choose = (file: File) => fireEvent.change(screen.getByLabelText("Picture file"), { target: { files: [file] } });

  it("shows it is sending, makes Add wait, then keeps the address the shop gave", async () => {
    const answer = held<Wire>();
    shop.upload.mockReturnValueOnce(answer.promise);
    openAdd("categories", "Categories");
    type("Name", "Women");
    choose(png());
    expect(await screen.findByText("Uploading…")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();
    answer.release(reply(201, { url: "/api/v1/uploads/abc.png" }));
    expect(await screen.findByRole("button", { name: "Replace" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    expect(document.querySelector("img")).toHaveAttribute("src", "/api/v1/uploads/abc.png");
    shop.create.mockResolvedValueOnce(reply(201, { id: "women", name: "Women" }));
    press("Add");
    await waitFor(() => expect(shop.create).toHaveBeenCalledTimes(1));
    expect(shop.create).toHaveBeenCalledWith("categories", { image: "/api/v1/uploads/abc.png", name: "Women" });
    expect(JSON.stringify(shop.create.mock.calls)).not.toContain("data:");
  });

  it("shows the shop's refusal beside the control and keeps the other values", async () => {
    shop.upload.mockResolvedValueOnce(reply(415, { error: "unsupported_media_type", message: "Choose a JPEG, PNG or WebP picture." }));
    openAdd("categories", "Categories");
    type("Name", "Women");
    choose(new File(["x"], "a.gif", { type: "image/gif" }));
    expect(await screen.findByText("Choose a JPEG, PNG or WebP picture.")).toHaveAttribute("role", "alert");
    expect(document.querySelector("img")).toBeNull();
    expect(screen.getByRole("button", { name: "Choose a picture" })).toBeEnabled();
    expect(screen.getByLabelText("Name")).toHaveValue("Women");

    shop.upload.mockResolvedValueOnce(reply(0));
    choose(png());
    expect(await screen.findByText("We could not upload that picture. Please try again.")).toBeInTheDocument();
    expect(screen.queryByText("Choose a JPEG, PNG or WebP picture.")).toBeNull();
  });
});

describe("the delete question", () => {
  const ask = (onClose = none, onDeleted = none) =>
    render(
      <MemoryRouter>
        <DeleteQuestion title="Delete Samba OG?" onClose={onClose} onDeleted={onDeleted}
          onDelete={() => deleteRow("products", "samba-og")} />
      </MemoryRouter>,
    );

  it("stays open with the shop's sentence inside it when the shop refuses", async () => {
    shop.remove.mockResolvedValueOnce(reply(409, { error: "conflict", message: "Orders refer to this shoe. Set it to draft instead." }));
    const gone = vi.fn();
    ask(none, gone);
    press("Delete");
    const dialog = screen.getByRole("alertdialog");
    expect(await within(dialog).findByText("Orders refer to this shoe. Set it to draft instead.")).toHaveAttribute("role", "alert");
    expect(within(dialog).getByText("Delete Samba OG?")).toBeInTheDocument();
    expect(gone).not.toHaveBeenCalled();
    expect(within(dialog).getByRole("button", { name: "Delete" })).toBeEnabled();
  });

  it("says it could not delete when the shop cannot be reached", async () => {
    shop.remove.mockResolvedValueOnce(reply(0));
    ask();
    press("Delete");
    expect(await screen.findByText("We could not delete. Please try again.")).toBeInTheDocument();
  });

  it("reports a delete the shop accepted, and sends nothing on Cancel", async () => {
    const closed = vi.fn();
    const gone = vi.fn();
    ask(closed, gone);
    press("Cancel");
    expect(shop.remove).not.toHaveBeenCalled();
    await waitFor(() => expect(closed).toHaveBeenCalled());
    cleanup();
    shop.remove.mockResolvedValueOnce(reply(204));
    ask(none, gone);
    press("Delete");
    await waitFor(() => expect(gone).toHaveBeenCalledTimes(1));
    expect(shop.remove).toHaveBeenCalledWith("products", "samba-og");
  });
});
