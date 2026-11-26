/**
 * ONE RECORD, OPEN.
 *
 * The other screen of a management area: what the list opens onto. A record
 * with four fields belongs in the list's dialog and never needs this; a
 * record with twenty — a product with its description, its pictures, its
 * price, its stock and its status — needs a page, and the fields group into
 * tabs (RecordForm).
 *
 *   export default function Product() {
 *     const { id } = useParams();
 *     return (
 *       <RecordPage collection="products" id={id} nav={ADMIN_NAV}
 *         backTo="/admin/products"
 *         title={useContent("pages.admin.products.title")} />
 *     );
 *   }
 *
 * The fields come from src/data/schema.json; giving a spec a `section` puts
 * it on a tab of that name, and a page that passes none gets one panel.
 *
 * WHAT IT REFUSES TO GUESS: a record the shop does not have. An id that is not
 * there is a link somebody followed after the record was deleted, or a typed
 * address, and a page that draws an empty form for it invites the person to
 * save a new record under an id of their own — so it says so and offers the
 * way back. It says so ONLY once the shop has answered: while the record is
 * being fetched, or when the shop could not be reached, the page says nothing
 * rather than say a record is missing that may not be.
 */
import { useNavigate } from "react-router-dom";
import DashboardLayout, { type NavGroup } from "../DashboardLayout";
import { Breadcrumb, Button, Empty } from "./index";
import RecordForm from "./RecordForm";
import { fieldsOf, type FieldSpec } from "./fields";
import { useRecord } from "../../data";

export default function RecordPage({
  collection, id, nav, title, brand, fields, backTo, remove = true, children,
}: {
  collection: string;
  /** From useParams(). */
  id: string | undefined;
  nav: readonly NavGroup[];
  /** The AREA's name for the top bar — "Products", not this record's name. */
  title: string;
  brand?: React.ReactNode;
  fields?: readonly FieldSpec[];
  /** The list this record belongs to, for the crumb and after a delete. */
  backTo?: string;
  remove?: boolean;
  /** Under the form: related lists, a log, whatever this record has. */
  children?: React.ReactNode;
}) {
  const navigate = useNavigate();
  const { item, loading, error } = useRecord(collection, id);
  const specs = fields ?? fieldsOf(collection);
  const gone = !loading && !error && !item;

  return (
    <DashboardLayout title={title} nav={nav} brand={brand}>
      {item && (
        <RecordForm key={item.id} collection={collection} record={item} specs={specs} title={title}
          backTo={backTo} remove={remove}>
          {children}
        </RecordForm>
      )}
      {gone && (
        <>
          {backTo && <Breadcrumb trail={[{ label: title, to: backTo }, { label: id ?? "" }]} />}
          <Empty
            title="This record is not here"
            note="It may have been deleted since the link was made."
            action={backTo ? <Button onClick={() => navigate(backTo)}>Back to the list</Button> : undefined}
          />
        </>
      )}
    </DashboardLayout>
  );
}
