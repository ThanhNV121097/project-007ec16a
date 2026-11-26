import DataTable, { type Column } from "../DataTable";
import { useCollection, type Filter, type Row } from "../../data";
import type { TableQuery } from "../../lib/table";
import { Button, Empty, RowMenu } from "./index";

export interface ListFilter {
  key: string;
  label: string;
  options: readonly (readonly [string, string])[];
}

/**
 * THE TABLE OF A LIST, read from the shop.
 *
 * Its own component so that the list can be read again after a save: the
 * screen gives it a new `key`, it mounts afresh and asks the shop for the
 * page. What a person sees after saving is what the shop holds, never what
 * was typed.
 */
export default function ListTable({ collection, t, filters, columns, caption, rowHref, onAdd, onEdit, onDelete }: {
  collection: string;
  t: TableQuery;
  filters: readonly ListFilter[];
  columns: readonly Column<Row>[];
  caption: string;
  rowHref?: (row: Row) => string;
  /** Where the empty list's "Add" goes; left out where records are not made here. */
  onAdd?: () => void;
  onEdit: (row: Row) => void;
  /** Left out where records are not deleted here. */
  onDelete?: (row: Row) => void;
}) {
  const where: Record<string, Filter | undefined> = {};
  for (const f of filters) where[f.key] = t.get(f.key);
  const { items, total, pages, page, loading } = useCollection(collection, {
    page: t.page, pageSize: t.size, sort: t.sort, q: t.q, where,
  });
  const addButton = onAdd ? <Button kind="primary" icon="plus" onClick={onAdd}>Add</Button> : null;

  return (
    <DataTable<Row>
      columns={columns}
      rows={items}
      rowKey={(r) => r.id}
      total={total}
      page={page}
      pages={pages}
      size={t.size}
      sort={t.sort}
      onSort={t.toggleSort}
      onPage={t.setPage}
      onSize={t.setSize}
      loading={loading}
      caption={caption}
      href={rowHref}
      actions={(row) => (
        <RowMenu
          items={[
            { label: "Edit", onClick: () => onEdit(row) },
            ...(onDelete ? [{ label: "Delete", onClick: () => onDelete(row), danger: true }] : []),
          ]}
        />
      )}
      empty={
        <Empty
          title={t.filtering ? "Nothing matches those filters" : "Nothing here yet"}
          note={t.filtering ? "Clear a filter to see more." : undefined}
          action={t.filtering ? <Button onClick={t.clear}>Clear filters</Button> : addButton}
        />
      }
    />
  );
}
