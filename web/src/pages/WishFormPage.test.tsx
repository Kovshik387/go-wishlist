import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../lib/api";
import { WishFormPage } from "./WishFormPage";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return {
    ...actual,
    api: {
      ...actual.api,
      createWish: vi.fn(),
    },
  };
});

describe("WishFormPage", () => {
  beforeEach(() => {
    vi.mocked(api.createWish).mockResolvedValue({
      id: "wish-1",
      wishlistId: "list-1",
      productUrl: "",
      title: "Наушники",
      description: "",
      imageUrl: "https://images.example/headphones.jpg",
      currency: "RUB",
      priority: "normal",
      quantity: 1,
      attributes: {},
      storeDomain: "",
      version: 1,
      isReserved: false,
      reservedByMe: false,
      createdAt: "2026-07-24T00:00:00Z",
      updatedAt: "2026-07-24T00:00:00Z",
    });
  });

  it("accepts an image URL and submits hidden defaults", async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={["/lists/list-1/wishes/new"]}>
          <Routes>
            <Route path="/lists/:id/wishes/new" element={<WishFormPage />} />
            <Route path="/lists/:id" element={<div>Желание добавлено</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.queryByText("Валюта")).not.toBeInTheDocument();
    expect(screen.queryByText("Кол-во")).not.toBeInTheDocument();
    expect(screen.queryByText("Насколько хочется")).not.toBeInTheDocument();
    expect(screen.queryByText("Размер, цвет и вариант")).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Название"), { target: { value: "Наушники" } });
    fireEvent.change(screen.getByLabelText("Ссылка на фото"), {
      target: { value: "https://images.example/headphones.jpg" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Добавить в список" }));

    await waitFor(() => expect(api.createWish).toHaveBeenCalled());
    expect(api.createWish).toHaveBeenCalledWith(
      "list-1",
      expect.objectContaining({
        imageUrl: "https://images.example/headphones.jpg",
        currency: "RUB",
        priority: "normal",
        quantity: 1,
        attributes: {},
      }),
    );
    expect(await screen.findByText("Желание добавлено")).toBeInTheDocument();
  });
});
