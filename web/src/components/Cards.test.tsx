import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import type { Wish } from "../types";
import { WishCard } from "./Cards";

const wish: Wish = {
  id: "wish-1",
  wishlistId: "list-1",
  productUrl: "https://www.ozon.ru/product/123",
  title: "Настольная игра",
  description: "Лучше русское издание: https://example.com/game.",
  imageUrl: "",
  currency: "RUB",
  priority: "normal",
  quantity: 1,
  attributes: {},
  storeDomain: "www.ozon.ru",
  version: 1,
  isReserved: false,
  reservedByMe: false,
  createdAt: "2026-07-24T00:00:00Z",
  updatedAt: "2026-07-24T00:00:00Z",
};

describe("WishCard", () => {
  it("shows and hides a wish comment", () => {
    const client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <WishCard wish={wish} owner />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.queryByText(/Лучше русское издание/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Показать комментарий" }));
    expect(screen.getByText(/Лучше русское издание/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "https://example.com/game" })).toHaveAttribute(
      "href",
      "https://example.com/game",
    );
    fireEvent.click(screen.getByRole("button", { name: "Скрыть комментарий" }));
    expect(screen.queryByText(/Лучше русское издание/)).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "www.ozon.ru" })).toHaveAttribute(
      "href",
      "https://www.ozon.ru/product/123",
    );
  });
});
