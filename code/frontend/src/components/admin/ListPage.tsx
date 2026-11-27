/**
 * THE LIST SCREEN OF A MANAGEMENT AREA, WHOLE.
 *
 * Nine tenths of every CMS this product builds is this one screen, repeated
 * per collection: a menu down the left, a search box and the filters the
 * list is really read through, a table with its order and its page numbers,
 * Add in a dialog, Edit on the row, Delete behind a question, and the empty
 * state a new account sees on its first day.
 *
 *   export default function Products() {
 *     return (
 *       <ListPage
 *         collection="products"
 *         title={useContent("pages.admin.products.title")}
 *         nav={ADMIN_NAV}
 *         rowHref={(r) => `/admin/products/${r.id}`}
 *         filters={[{ key: "status", label: "Status",
 *           options: [["draft", "Draft"], ["live", "Live"]] }]}
 *       />
 *     );
 *   }
 *
 * Nothing above is required but the collection, the title and the menu: the
 * columns and the form come from src/data/schema.json (fields.tsx), and
 * `columns` or `fields` replace them where a turn wants something else.
 *
 * THE SHOP HAS THE LAST WORD. Add, Edit and Delete go to the shop and wait
 * for it (RecordDialog, DeleteQuestion); when it accepts, the table is read
 * again and shows what the shop holds, never what was typed.
 *
 * THE FILTERS AND THE PAGE LIVE IN THE ADDRESS BAR (src/lib/table.ts), so
 * "what is waiting for review" is a link somebody can send, and opening a
 * record and coming back lands on the same page of the same list.
 */
import { useState } from "react";
import DashboardLayout, { type NavGroup } from "../DashboardLayout";
import { type Column } from "../DataTable";
import TableFilters, { SearchBox, FilterSelect } from "../TableFilters";
import { Button } from "./index";
import DeleteQuestion from "./DeleteQuestion";
import ListTable, { type ListFilter } from "./ListTable";
import RecordDialog from "./RecordDialog";
import { columnsFromFields, fieldsOf, titleOf, type FieldSpec } from "./fields";
import { deleteRow } from "./store";
import { type Row } from "../../data";
import { useTableQuery } from "../../lib/table";

export type { ListFilter } from "./ListTable";

export default function ListPage({
  collection, title, nav, brand, columns, fields, filters = [], search = true,
  searchPlaceholder, rowHref, add = true, remove = true, caption, actions, children,
}: {
  /** The collection this screen lists, as src/data/schema.json names it. */
  collection: string;
  title: string;
  nav: readonly NavGroup[];
  brand?: React.ReactNode;
  /** Replaces the columns derived from the schema. */
  columns?: readonly Column<Row>[];
  /** Replaces the form fields derived from the schema. */
  fields?: readonly FieldSpec[];
  filters?: readonly ListFilter[];
  search?: boolean;
  searchPlaceholder?: string;
  /** Where a row opens. Without it the row does not open anywhere. */
  rowHref?: (row: Row) => string;
  /** False where records are not created here — an orders list. */
  add?: boolean;
  remove?: boolean;
  caption?: string;
  /** Beside Add in the top bar: Export, Import, whatever this list has. */
  actions?: React.ReactNode;
  /** Above the table: a row of <Stat>, a notice, anything this list needs. */
  children?: React.ReactNode;
}) {
  const t = useTableQuery({ size: 20 });
  const specs = fields ?? fieldsOf(collection);

  // The record in the dialog: a row being edited, or a blank one to add.
  const [editing, setEditing] = useState<Row | null>(null);
  const [confirming, setConfirming] = useState<Row | null>(null);
  // Bumped after a save or a delete: the table is mounted afresh and reads the shop.
  const [version, setVersion] = useState(0);
  const open = (row: Row | null) => setEditing(row ?? ({ id: "" } as Row));
  const openAdd = () => open(null);
  const reread = () => setVersion((v) => v + 1);

  const addButton = add ? <Button kind="primary" icon="plus" onClick={openAdd}>Add</Button> : null;

  return (
    <DashboardLayout title={title} nav={nav} brand={brand} actions={<>{actions}{addButton}</>}>
      {children}
      {(search || filters.length > 0) && (
        <TableFilters onClear={t.filtering ? t.clear : undefined}>
          {search && <SearchBox value={t.q} onChange={t.setQ} placeholder={searchPlaceholder} />}
          {filters.map((f) => (
            <FilterSelect
              key={f.key}
              label={f.label}
              value={t.get(f.key)}
              onChange={(v) => t.set(f.key, v)}
              options={f.options}
            />
          ))}
        </TableFilters>
      )}

      <ListTable
        key={version}
        collection={collection}
        t={t}
        filters={filters}
        columns={columns ?? columnsFromFields(specs)}
        caption={caption ?? title}
        rowHref={rowHref}
        onAdd={add ? openAdd : undefined}
        onEdit={open}
        onDelete={remove ? setConfirming : undefined}
      />

      {/* Keyed on the record, so opening another one starts from ITS values
          and not from what was typed into the last. */}
      <RecordDialog key={editing?.id ?? "closed"} collection={collection} specs={specs} record={editing}
        heading={title} onClose={() => setEditing(null)}
        onSaved={() => { setEditing(null); reread(); }} />

      {confirming && (
        <DeleteQuestion
          key={confirming.id}
          title={`Delete ${titleOf(confirming)}?`}
          onClose={() => setConfirming(null)}
          onDelete={() => deleteRow(collection, confirming.id)}
          onDeleted={() => { setConfirming(null); reread(); }}
        />
      )}
    </DashboardLayout>
  );
}
