import { type ReactNode } from "react";
import { Navigate, Outlet, Route, Routes } from "react-router-dom";
import ContactDock from "./components/ContactDock";
import { Motion } from "./motion";
import SiteFooter from "./components/SiteFooter";
import SiteHeader from "./components/SiteHeader";
import { useSession } from "./lib/session";
import About from "./pages/About";
import AdminKinds from "./pages/AdminKinds";
import AdminLines from "./pages/AdminLines";
import AdminOrder from "./pages/AdminOrder";
import AdminOrders from "./pages/AdminOrders";
import AdminOverview from "./pages/AdminOverview";
import AdminProduct from "./pages/AdminProduct";
import AdminProducts from "./pages/AdminProducts";
import AdminReviews from "./pages/AdminReviews";
import AdminSignIn from "./pages/AdminSignIn";
import AdminStaff from "./pages/AdminStaff";
import Cart from "./pages/Cart";
import Checkout from "./pages/Checkout";
import Contact from "./pages/Contact";
import Home from "./pages/Home";
import Order from "./pages/Order";
import Product from "./pages/Product";
import Returns from "./pages/Returns";
import Shipping from "./pages/Shipping";
import Shop from "./pages/Shop";
import SizeChart from "./pages/SizeChart";
import Track from "./pages/Track";

/**
 * Two sites in one address: the shop a customer visits, and the side the
 * dealer runs it from. They do not share a shell: a customer's page sits
 * between the hairline header and the footer; a management page sits in
 * <DashboardLayout>, which each of them brings with it.
 */
function Shell() {
  return (
    <div className="min-h-screen overflow-x-clip bg-ground font-body text-ink">
      {/* The motion the owner chose, on every page (src/motion). */}
      <Motion />
      <SiteHeader />
      <main className="mx-auto max-w-page px-[var(--gutter)]"><Outlet /></main>
      <SiteFooter />
      <ContactDock />
    </div>
  );
}

/** The management area's door, through the scaffold's own session (src/lib/session.ts). */
function SignedIn({ children }: { children: ReactNode }) {
  return useSession() ? <>{children}</> : <Navigate to="/admin/sign-in" replace />;
}

export default function App() {
  return (
    <Routes>
      <Route element={<Shell />}>
        <Route path="/" element={<Home />} />
        <Route path="/shop" element={<Shop />} />
        <Route path="/shop/:id" element={<Product />} />
        <Route path="/size-chart" element={<SizeChart />} />
        <Route path="/cart" element={<Cart />} />
        <Route path="/checkout" element={<Checkout />} />
        <Route path="/orders/:id" element={<Order />} />
        <Route path="/track" element={<Track />} />
        <Route path="/shipping" element={<Shipping />} />
        <Route path="/returns" element={<Returns />} />
        <Route path="/about" element={<About />} />
        <Route path="/contact" element={<Contact />} />
      </Route>
      <Route path="/admin/sign-in" element={<AdminSignIn />} />
      <Route path="/admin" element={<SignedIn><AdminOverview /></SignedIn>} />
      <Route path="/admin/orders" element={<SignedIn><AdminOrders /></SignedIn>} />
      <Route path="/admin/orders/:id" element={<SignedIn><AdminOrder /></SignedIn>} />
      <Route path="/admin/products" element={<SignedIn><AdminProducts /></SignedIn>} />
      <Route path="/admin/products/:id" element={<SignedIn><AdminProduct /></SignedIn>} />
      <Route path="/admin/lines" element={<SignedIn><AdminLines /></SignedIn>} />
      <Route path="/admin/categories" element={<SignedIn><AdminKinds /></SignedIn>} />
      <Route path="/admin/reviews" element={<SignedIn><AdminReviews /></SignedIn>} />
      <Route path="/admin/staff" element={<SignedIn><AdminStaff /></SignedIn>} />
    </Routes>
  );
}
