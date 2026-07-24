import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../lib/api";
import { WishlistFormPage } from "./WishlistFormPage";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return {
    ...actual,
    api: {
      ...actual.api,
      createWishlist: vi.fn(),
      upload: vi.fn(),
    },
  };
});

describe("WishlistFormPage", () => {
  afterEach(cleanup);

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.upload).mockResolvedValue({ publicUrl: "/media/user-1/cover.jpg" });
    vi.mocked(api.createWishlist).mockResolvedValue({
      id: "list-1",
      ownerId: "user-1",
      title: "Мой день рождения",
      description: "",
      emoji: "🎁",
      coverUrl: "",
      occasion: "birthday",
      visibility: "public",
      allowReservations: true,
      ownerSeesReservations: false,
      publicToken: "public",
      version: 1,
      wishCount: 0,
      createdAt: "2026-07-24T00:00:00Z",
      updatedAt: "2026-07-24T00:00:00Z",
    });
  });

  it("uploads and submits a wishlist cover", async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={["/lists/new"]}>
          <Routes>
            <Route path="/lists/new" element={<WishlistFormPage />} />
            <Route path="/lists/:id" element={<div>Список создан</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const image = new File(["image"], "cover.jpg", { type: "image/jpeg" });
    fireEvent.change(screen.getByLabelText("Фотография списка"), { target: { files: [image] } });
    await waitFor(() => expect(vi.mocked(api.upload).mock.calls[0]?.[0]).toBe(image));
    expect(await screen.findByAltText("Обложка списка")).toHaveAttribute("src", "/media/user-1/cover.jpg");

    fireEvent.change(screen.getByLabelText("Название"), { target: { value: "Мой список" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать список" }));

    await waitFor(() => expect(api.createWishlist).toHaveBeenCalledWith(
      expect.objectContaining({ coverUrl: "/media/user-1/cover.jpg" }),
    ));
  });

  it("validates and submits a new wishlist", async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={["/lists/new"]}>
          <Routes>
            <Route path="/lists/new" element={<WishlistFormPage />} />
            <Route path="/lists/:id" element={<div>Список создан</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Создать список" }));
    expect(await screen.findByText("Напишите название")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Название"), {
      target: { value: "Мой день рождения" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Создать список" }));

    await waitFor(() => expect(api.createWishlist).toHaveBeenCalled());
    expect(await screen.findByText("Список создан")).toBeInTheDocument();
  });
});
