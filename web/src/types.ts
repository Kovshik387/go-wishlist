export type User = {
  id: string;
  telegramId?: number;
  username: string;
  displayName: string;
  avatarUrl: string;
  languageCode: string;
  timezone: string;
  botWriteAllowed: boolean;
  onboardingCompleted: boolean;
  createdAt: string;
};

export type WishlistVisibility = "public" | "link" | "private";

export type Wishlist = {
  id: string;
  ownerId: string;
  title: string;
  description: string;
  emoji: string;
  coverUrl: string;
  occasion: "birthday" | "wedding" | "new_year" | "housewarming" | "other";
  eventDate?: string;
  visibility: WishlistVisibility;
  allowReservations: boolean;
  ownerSeesReservations: boolean;
  publicToken?: string;
  version: number;
  wishCount: number;
  createdAt: string;
  updatedAt: string;
  owner?: User;
  wishes?: Wish[];
};

export type Wish = {
  id: string;
  wishlistId: string;
  productUrl: string;
  title: string;
  description: string;
  imageUrl: string;
  priceMinor?: number;
  currency: string;
  priority: "normal" | "high";
  quantity: number;
  attributes: Record<string, string>;
  storeDomain: string;
  version: number;
  isReserved: boolean;
  reservedByMe: boolean;
  reservedBy?: User;
  createdAt: string;
  updatedAt: string;
  wishlist?: Wishlist;
  author?: User;
};

export type Reservation = {
  id: string;
  wishId: string;
  status: string;
  createdAt: string;
  wish: Wish;
};

export type NotificationPreferences = {
  enabled: boolean;
  newWishes: boolean;
  newWishlists: boolean;
  eventReminders: boolean;
  reservationUpdates: boolean;
  quietHoursEnabled: boolean;
  quietStart: string;
  quietEnd: string;
};

export type BrandConfig = {
  name: string;
  emoji: string;
  primary: string;
  accent: string;
  botUsername: string;
  appShortName: string;
};
