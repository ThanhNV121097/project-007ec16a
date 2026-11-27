// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
const stubs = vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
  return { post: vi.fn() };
});

// The API client's one write, standing in for POST /v1/order-lookups; the real ApiError stays.
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: { ...actual.api, post: stubs.post } };
});

import TrackForm from "../components/TrackForm";
import { ApiError } from "../lib/api";

// The Track form: what it sends, what each answer is told in, and what the
// browser keeps. The shop's answer is stubbed at the transport.

const post = stubs.post;
const FOUND = { id: "doi-24817", number: "DOI-24817", placed_at: "2026-11-24T09:12:00+07:00", token: "q3J0n8Zk2LmPa7Vx9TrBwE" };
const NOT_FOUND = "No order matches that number and phone. Check both, or message us.";
const TOO_MANY = "Too many tries. Please wait 15 minutes, or message us with your order number.";
const OFFLINE = "You seem to be offline. Check your connection and try again.";
const FAILED = "We could not look that up. Please try again.";

beforeEach(() => {
  localStorage.clear();
  post.mockReset();
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function open() {
  render(
    <MemoryRouter initialEntries={["/track"]}>
      <Routes>
        <Route path="/track" element={<TrackForm />} />
        <Route path="/orders/:id" element={<p>the order page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });
const button = () => screen.getByTestId("track-submit");
const press = () => fireEvent.click(button());

describe("a right pair", () => {
  it("sends what was typed, as typed, and opens the order", async () => {
    post.mockResolvedValueOnce(FOUND);
    open();
    type("Order number", "doi 24817");
    type("Phone", "+84938410552");
    press();
    expect(await screen.findByText("the order page")).toBeInTheDocument();
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("order-lookups", { number: "doi 24817", phone: "+84938410552" });
  });

  it("keeps the id, number, placed time and token in the one store and nothing else", async () => {
    const answered = { ...FOUND, phone: "0938410552", customer: "Lan Nguyen" };
    post.mockResolvedValueOnce(answered);
    open();
    type("Order number", "DOI-24817");
    type("Phone", "0938 410 552");
    press();
    await screen.findByText("the order page");
    expect(JSON.parse(localStorage.getItem("doi:placed") ?? "null")).toEqual([FOUND]);
    expect(localStorage.length).toBe(1);
    expect(localStorage.getItem("doi:placed")).not.toMatch(/0938|Lan/);
  });

  it("holds one entry for an order looked up twice, with the newest token", async () => {
    post.mockResolvedValueOnce(FOUND).mockResolvedValueOnce({ ...FOUND, token: "second-token-xxxxxxxxxxxx" });
    for (let i = 0; i < 2; i++) {
      open();
      type("Order number", "24817");
      type("Phone", "0938410552");
      press();
      await screen.findByText("the order page");
      cleanup();
    }
    expect(JSON.parse(localStorage.getItem("doi:placed") ?? "null")).toEqual([{ ...FOUND, token: "second-token-xxxxxxxxxxxx" }]);
  });

  it("still opens the order when the browser refuses storage", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("storage is full");
    });
    post.mockResolvedValueOnce(FOUND);
    open();
    press();
    expect(await screen.findByText("the order page")).toBeInTheDocument();
    expect(screen.queryByTestId("track-error")).toBeNull();
  });

  it("is a failed lookup when the answer is not the four things a right pair gives", async () => {
    post.mockResolvedValueOnce({ id: "../x", number: "DOI-1", placed_at: "2026-11-24T09:12:00+07:00", token: "x" });
    open();
    press();
    expect((await screen.findByTestId("track-error")).textContent).toBe(FAILED);
    expect(localStorage.getItem("doi:placed")).toBeNull();
  });
});

describe("each way it can fail has its own sentence, in the same place", () => {
  it.each([
    ["a pair that does not match (404 not_found)", new ApiError(404, "not_found", "Not found."), NOT_FOUND],
    ["too many tries, whatever its words (429)", new ApiError(429, "internal", "Slow down"), TOO_MANY],
    ["a 404 that is not the shop's own", new ApiError(404, "internal", "x"), FAILED],
    ["a 500", new ApiError(500, "internal", "x"), FAILED],
    ["a 502", new ApiError(502, "internal", "x"), FAILED],
    ["a shop that cannot be reached while online", new ApiError(0, "network", "x"), FAILED],
    ["something that is not an ApiError", new TypeError("boom"), FAILED],
  ])("%s", async (_name, failure, sentence) => {
    post.mockRejectedValueOnce(failure);
    open();
    type("Order number", "DOI-24817");
    type("Phone", "0938 410 553");
    press();
    const line = await screen.findByTestId("track-error");
    expect(line.textContent).toBe(sentence);
    expect(line).toHaveAttribute("role", "alert");
    expect(line.querySelector("svg")).not.toBeNull();
    expect(screen.getAllByTestId("track-error")).toHaveLength(1);
    expect(screen.getByLabelText("Order number")).toHaveValue("DOI-24817");
    expect(screen.getByLabelText("Phone")).toHaveValue("0938 410 553");
    expect(button()).toBeEnabled();
    expect(button()).toHaveTextContent("Find my order");
    expect(post).toHaveBeenCalledTimes(1);
  });

  it("draws every not-found alike: a wrong phone, an unknown number and empty fields", async () => {
    const lines: string[] = [];
    for (const [number, phone] of [["DOI-24817", "0938 410 553"], ["DOI-99999", "0938 410 552"], ["DOI-24817", ""], ["", ""]]) {
      post.mockRejectedValueOnce(new ApiError(404, "not_found", "Not found."));
      cleanup();
      open();
      type("Order number", number);
      type("Phone", phone);
      press();
      lines.push((await screen.findByTestId("track-error")).outerHTML);
    }
    expect(new Set(lines).size).toBe(1);
    expect(post).toHaveBeenLastCalledWith("order-lookups", { number: "", phone: "" });
  });

  it("says it is offline, and sends nothing", async () => {
    vi.spyOn(window.navigator, "onLine", "get").mockReturnValue(false);
    open();
    type("Order number", "DOI-24817");
    type("Phone", "0938 410 552");
    press();
    expect((await screen.findByTestId("track-error")).textContent).toBe(OFFLINE);
    expect(post).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Phone")).toHaveValue("0938 410 552");
    expect(button()).toBeEnabled();
  });

  it("says it is offline when the connection fails and the browser has gone offline meanwhile", async () => {
    post.mockImplementationOnce(() => {
      vi.spyOn(window.navigator, "onLine", "get").mockReturnValue(false);
      return Promise.reject(new ApiError(0, "network", "x"));
    });
    open();
    press();
    expect((await screen.findByTestId("track-error")).textContent).toBe(OFFLINE);
  });
});

describe("while the shop answers", () => {
  it("disables the button, says Searching…, keeps the values and sends one request", async () => {
    let refuse: (e: unknown) => void = () => undefined;
    post.mockReturnValueOnce(new Promise((_, reject) => (refuse = reject)));
    open();
    type("Order number", "DOI-24817");
    type("Phone", "0938 410 553");
    press();
    await waitFor(() => expect(button()).toBeDisabled());
    expect(button()).toHaveTextContent("Searching…");
    expect(screen.getByLabelText("Order number")).toHaveValue("DOI-24817");
    expect(screen.getByLabelText("Phone")).toHaveValue("0938 410 553");
    fireEvent.submit(button().closest("form") as HTMLFormElement);
    fireEvent.click(button());
    expect(post).toHaveBeenCalledTimes(1);
    refuse(new ApiError(404, "not_found", "Not found."));
    expect((await screen.findByTestId("track-error")).textContent).toBe(NOT_FOUND);
    expect(button()).toBeEnabled();
    expect(button()).toHaveTextContent("Find my order");
  });

  it("removes the old message when a new try starts", async () => {
    post.mockRejectedValueOnce(new ApiError(404, "not_found", "Not found."));
    open();
    press();
    await screen.findByTestId("track-error");
    post.mockReturnValueOnce(new Promise(() => undefined));
    press();
    await waitFor(() => expect(button()).toBeDisabled());
    expect(screen.queryByTestId("track-error")).toBeNull();
  });

  it("does not go to the order when the visitor has left the page by the time it answers", async () => {
    let answer: (v: typeof FOUND) => void = () => undefined;
    post.mockReturnValueOnce(new Promise((resolve) => (answer = resolve)));
    open();
    press();
    await waitFor(() => expect(button()).toBeDisabled());
    cleanup();
    answer(FOUND);
    await waitFor(() => expect(localStorage.getItem("doi:placed")).not.toBeNull());
    expect(screen.queryByText("the order page")).toBeNull();
  });
});
