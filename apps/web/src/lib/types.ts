// Shapes returned by the gateway (see the Go services' DTOs).

export type Category = "pastry" | "flower";

export type Product = {
  id: number;
  name: string;
  category: Category;
  description: string;
  priceIdr: number;
  imageUrl: string;
};

/** A product plus its stock; null when the inventory has no record for it. */
export type ProductWithStock = Product & { stock: number | null };

export type OrderStatus = "pending" | "awaiting_payment" | "confirmed" | "cancelled";

export type OrderItem = {
  productId: number;
  name: string;
  unitPriceIdr: number;
  quantity: number;
};

export type Order = {
  id: string;
  status: OrderStatus;
  cancelReason?: string;
  totalIdr: number;
  createdAt: string;
  items: OrderItem[];
};

export type Role = "customer" | "admin";

export type ProviderMode = "succeed" | "decline" | "down";
