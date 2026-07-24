import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import type { Wish } from "../types";
import { WishCard } from "./Cards";

const wish: Wish = {
  id: "wish-1",
  wishlistId: "list-1",
  productUrl: "",
  title: "Настольная игра",
  description: "Лучше русское издание, если оно есть.",
  imageUrl: "",
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

    expect(screen.queryByText(wish.description)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Показать комментарий" }));
    expect(screen.getByText(wish.description)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Скрыть комментарий" }));
    expect(screen.queryByText(wish.description)).not.toBeInTheDocument();
  });
});
