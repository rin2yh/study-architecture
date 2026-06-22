import { useCart } from "@/entities/cart";
import { PageLoading } from "@/shared/ui/page-loading";
import { CartList } from "./cart-list";
import { EmptyCart } from "./empty-cart";

export function CartPage() {
  const cart = useCart();

  if (!cart.ready) return <PageLoading />;
  if (cart.items.length === 0) return <EmptyCart />;
  return <CartList items={cart.items} onSetQty={cart.setQty} onRemove={cart.remove} />;
}
