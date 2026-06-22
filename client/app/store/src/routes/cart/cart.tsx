import { useCart } from "@/entities/cart";
import { PageLoading } from "@/shared/ui/page-loading";
import { CartList } from "./components/cart-list";
import { EmptyCart } from "./components/empty-cart";

export default function Cart() {
  const cart = useCart();

  if (!cart.ready) return <PageLoading />;
  if (cart.items.length === 0) return <EmptyCart />;
  return <CartList items={cart.items} onSetQty={cart.setQty} onRemove={cart.remove} />;
}
